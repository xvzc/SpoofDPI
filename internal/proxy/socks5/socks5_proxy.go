package socks5

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/dns"
	"github.com/xvzc/SpoofDPI/internal/logging"
	"github.com/xvzc/SpoofDPI/internal/matcher"
	"github.com/xvzc/SpoofDPI/internal/netutil"
	"github.com/xvzc/SpoofDPI/internal/proto"
	"github.com/xvzc/SpoofDPI/internal/proxy"
	"github.com/xvzc/SpoofDPI/internal/proxy/http"
	"github.com/xvzc/SpoofDPI/internal/ptr"
	"github.com/xvzc/SpoofDPI/internal/session"
)

type SOCKS5Proxy struct {
	logger zerolog.Logger

	resolver     dns.Resolver
	httpsHandler *http.HTTPSHandler // SOCKS5 primarily uses CONNECT, so we leverage HTTPSHandler
	ruleMatcher  matcher.RuleMatcher
	serverOpts   *config.ServerOptions
	policyOpts   *config.PolicyOptions
}

func NewSOCKS5Proxy(
	logger zerolog.Logger,
	resolver dns.Resolver,
	httpsHandler *http.HTTPSHandler,
	ruleMatcher matcher.RuleMatcher,
	serverOpts *config.ServerOptions,
	policyOpts *config.PolicyOptions,
) proxy.ProxyServer {
	return &SOCKS5Proxy{
		logger:       logger,
		resolver:     resolver,
		httpsHandler: httpsHandler,
		ruleMatcher:  ruleMatcher,
		serverOpts:   serverOpts,
		policyOpts:   policyOpts,
	}
}

func (p *SOCKS5Proxy) ListenAndServe(ctx context.Context, wait chan struct{}) {
	<-wait

	logger := p.logger.With().Ctx(ctx).Logger()

	// Using ListenTCP to match HTTPProxy style, though net.Listen is also fine
	listener, err := net.ListenTCP("tcp", p.serverOpts.ListenAddr)
	if err != nil {
		p.logger.Fatal().
			Err(err).
			Msgf("error creating socks5 listener on %s", p.serverOpts.ListenAddr.String())
	}

	logger.Info().
		Msgf("created a socks5 listener on %s", p.serverOpts.ListenAddr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			p.logger.Error().
				Err(err).
				Msg("failed to accept new connection")
			continue
		}

		go p.handleConnection(session.WithNewTraceID(context.Background()), conn)
	}
}

func (p *SOCKS5Proxy) handleConnection(ctx context.Context, conn net.Conn) {
	logger := logging.WithLocalScope(ctx, p.logger, "socks5")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer netutil.CloseConns(conn)

	// 1. Negotiation Phase
	if err := p.negotiate(conn); err != nil {
		logger.Debug().Err(err).Msg("socks5 negotiation failed")
		return
	}

	// 2. Request Phase
	req, err := proto.ReadSocks5Request(conn)
	if err != nil {
		if err != io.EOF {
			logger.Warn().Err(err).Msg("failed to read socks5 request")
		}
		return
	}

	// Only support CONNECT for now
	if req.Cmd != proto.CmdConnect {
		_ = proto.SOCKS5CommandNotSupportedResponse().Write(conn)
		logger.Warn().Uint8("cmd", req.Cmd).Msg("unsupported socks5 command")
		return
	}

	// Setup Logging Context
	remoteInfo := req.Domain
	if remoteInfo == "" {
		remoteInfo = req.IP.String()
	}
	ctx = session.WithHostInfo(ctx, remoteInfo)
	logger = logger.With().Ctx(ctx).Logger()

	logger.Debug().
		Str("from", conn.RemoteAddr().String()).
		Msg("new socks5 request")

	// 3. Match Domain Rules (if domain provided)
	var nameMatch *config.Rule
	if req.Domain != "" {
		nameMatch = p.ruleMatcher.Search(
			&matcher.Selector{
				Kind:   matcher.MatchKindDomain,
				Domain: ptr.FromValue(req.Domain),
			},
		)
		if nameMatch != nil && logger.GetLevel() == zerolog.TraceLevel {
			jsonAttrs, _ := json.Marshal(nameMatch)
			logger.Trace().RawJSON("values", jsonAttrs).Msg("name match")
		}
	}

	// 4. DNS Resolution
	// SOCKS5 allows IP or Domain. If Domain, we resolve. If IP, we use it directly.
	t1 := time.Now()
	var addrs []net.IPAddr

	if req.Domain != "" {
		// Resolve Domain
		rSet, err := p.resolver.Resolve(ctx, req.Domain, nil, nameMatch)
		if err != nil {
			_ = proto.SOCKS5FailureResponse().Write(conn)
			logging.ErrorUnwrapped(&logger, "dns lookup failed", err)
			return
		}
		addrs = rSet.Addrs
	} else {
		// IP Request - Just wrap the IP
		addrs = []net.IPAddr{{IP: req.IP}}
	}

	dt := time.Since(t1).Milliseconds()
	logger.Debug().
		Int("cnt", len(addrs)).
		Str("took", fmt.Sprintf("%dms", dt)).
		Msg("dns lookup ok")

	// Avoid recursively querying self.
	ok, err := validateDestination(addrs, req.Port, p.serverOpts.ListenAddr)
	if err != nil {
		logger.Debug().Err(err).Msg("error determining if valid destination")
		if !ok {
			_ = proto.SOCKS5FailureResponse().Write(conn)
			return
		}
	}

	// 6. Match IP Rules
	var selectors []*matcher.Selector
	for _, v := range addrs {
		selectors = append(selectors, &matcher.Selector{
			Kind: matcher.MatchKindAddr,
			IP:   ptr.FromValue(v.IP),
			Port: ptr.FromValue(uint16(req.Port)),
		})
	}

	addrMatch := p.ruleMatcher.SearchAll(selectors)
	if addrMatch != nil && logger.GetLevel() == zerolog.TraceLevel {
		jsonAttrs, _ := json.Marshal(addrMatch)
		logger.Trace().RawJSON("values", jsonAttrs).Msg("addr match")
	}

	bestMatch := matcher.GetHigherPriorityRule(addrMatch, nameMatch)
	if bestMatch != nil && *bestMatch.Block {
		logger.Debug().Msg("request is blocked by policy")
		_ = proto.SOCKS5FailureResponse().Write(conn)
		// Or specific code for blocked
		return
	}

	// 7. Handover to Handler
	// Important: We must send a success reply to the client BEFORE handing over to the handler,
	// because the handler (SpoofDPI) typically expects a raw stream where it can start TLS handshake immediately.
	// However, standard SOCKS5 expects the proxy to connect to the target FIRST, then send success.
	// Since SpoofDPI handler does the connection, we might need to send success here optimistically.

	// [Optimistic Success]
	// We tell the client "OK, we are connected" so it starts sending data (e.g. ClientHello).
	// The real connection happens inside p.tcpHandler.HandleRequest.
	// Note: BIND addr/port is usually 0.0.0.0:0 if we don't care.
	if err := proto.SOCKS5SuccessResponse().Bind(net.IPv4zero).Port(0).Write(conn); err != nil {
		logger.Error().Err(err).Msg("failed to write socks5 success reply")
		return
	}

	dst := &http.Destination{
		Domain:  req.Domain,
		Addrs:   addrs,
		Port:    req.Port,
		Timeout: *p.serverOpts.Timeout,
	}

	// Note: 'req' is nil here because it's not an HTTP request yet.
	// The handler must be able to handle nil req or we wrap a dummy one.
	// Assuming Handler is adapted to deal with raw streams or nil reqs for SOCKS mode.
	handleErr := p.httpsHandler.HandleRequest(ctx, conn, dst, bestMatch)

	if handleErr == nil {
		return
	}

	logger.Warn().Err(handleErr).Msg("error handling request")
	if !errors.Is(handleErr, netutil.ErrBlocked) {
		return
	}

	// 8. Auto Config (Duplicate logic from HTTPProxy)
	if nameMatch != nil {
		logger.Info().
			Interface("match", nameMatch.Match.Domains).
			Str("name", *nameMatch.Name).
			Msg("skipping auto-config (duplicate policy)")
		return
	}

	if addrMatch != nil {
		logger.Info().
			Interface("match", addrMatch.Match.Addrs).
			Str("name", *addrMatch.Name).
			Msg("skipping auto-config (duplicate policy)")
		return
	}

	if *p.policyOpts.Auto && p.policyOpts.Template != nil {
		newRule := p.policyOpts.Template.Clone()
		targetDomain := req.Domain
		if targetDomain == "" && len(addrs) > 0 {
			// If request was by IP, we can't really add a domain rule,
			// maybe add IP rule or skip. Use domain if available.
			targetDomain = addrs[0].IP.String()
		}

		newRule.Match = &config.MatchAttrs{Domains: []string{targetDomain}}

		if err := p.ruleMatcher.Add(newRule); err != nil {
			logger.Info().Err(err).Msg("failed to add config automatically")
		} else {
			logger.Info().Msg("automatically added to config")
		}
	}
}

// validateDestination checks if we are recursively querying ourselves.
// This function needs to be duplicated or moved to a shared utility if common logic is identical.
// For now, I'll use a local helper or assume the logic is specific enough.
// Since `isRecursiveDst` was a method on HTTPProxy, I'll assume similar logic is needed here.
// I'll implement a local version for now to keep it self-contained in this package or duplicated from http/proxy.go.
// Or I can just omit it if I don't want to duplicate, but safety is important.
// I'll add a simple check against listening port/loopback.
func validateDestination(
	dstAddrs []net.IPAddr,
	dstPort int,
	listenAddr *net.TCPAddr,
) (bool, error) {
	if dstPort != int(listenAddr.Port) {
		return true, nil
	}

	for _, dstAddr := range dstAddrs {
		ip := dstAddr.IP
		if ip.IsLoopback() {
			return false, nil
		}

		ifAddrs, err := net.InterfaceAddrs()
		if err != nil {
			return false, err
		}

		for _, addr := range ifAddrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ipnet.IP.Equal(ip) {
					return false, nil
				}
			}
		}
	}
	return true, nil
}

// negotiate performs SOCKS5 auth negotiation (NoAuth only).
func (p *SOCKS5Proxy) negotiate(conn net.Conn) error {
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return err
	}

	if header[0] != proto.SOCKSVersion {
		return fmt.Errorf("unsupported version: %d", header[0])
	}

	nMethods := int(header[1])
	methods := make([]byte, nMethods)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return err
	}

	// Respond: Version 5, Method NoAuth(0)
	_, err := conn.Write([]byte{proto.SOCKSVersion, proto.AuthNone})
	return err
}
