package proxy

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
	"github.com/xvzc/SpoofDPI/internal/handler"
	"github.com/xvzc/SpoofDPI/internal/logging"
	"github.com/xvzc/SpoofDPI/internal/matcher"
	"github.com/xvzc/SpoofDPI/internal/netutil"
	"github.com/xvzc/SpoofDPI/internal/proto"
	"github.com/xvzc/SpoofDPI/internal/ptr"
	"github.com/xvzc/SpoofDPI/internal/session"
)

type HTTPProxy struct {
	logger zerolog.Logger

	resolver     dns.Resolver
	httpHandler  handler.RequestHandler
	httpsHandler handler.RequestHandler
	ruleMatcher  matcher.RuleMatcher
	serverOpts   *config.ServerOptions
	policyOpts   *config.PolicyOptions
}

func NewHTTPProxy(
	logger zerolog.Logger,
	resolver dns.Resolver,
	httpHandler handler.RequestHandler,
	httpsHandler handler.RequestHandler,
	ruleMatcher matcher.RuleMatcher,
	serverOpts *config.ServerOptions,
	policyOpts *config.PolicyOptions,
) Proxy {
	return &HTTPProxy{
		logger:       logger,
		resolver:     resolver,
		httpHandler:  httpHandler,
		httpsHandler: httpsHandler,
		ruleMatcher:  ruleMatcher,
		serverOpts:   serverOpts,
		policyOpts:   policyOpts,
	}
}

func (p *HTTPProxy) ListenAndServe(ctx context.Context, wait chan struct{}) {
	<-wait

	logger := p.logger.With().Ctx(ctx).Logger()

	listener, err := net.ListenTCP("tcp", p.serverOpts.ListenAddr)
	if err != nil {
		p.logger.Fatal().
			Err(err).
			Msgf("error creating listener on %s", p.serverOpts.ListenAddr.String())
	}

	logger.Info().
		Msgf("created a listener on %s", p.serverOpts.ListenAddr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			p.logger.Error().
				Err(err).
				Msgf("failed to accept new connection")

			continue
		}

		go p.handleConnection(session.WithNewTraceID(context.Background()), conn)
	}
}

func (p *HTTPProxy) handleConnection(ctx context.Context, conn net.Conn) {
	logger := logging.WithLocalScope(ctx, p.logger, "conn")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer netutil.CloseConns(conn)

	req, err := proto.ReadHttpRequest(conn)
	if err != nil {
		if err != io.EOF {
			logger.Warn().Err(err).Msg("failed to read http request")
		}

		return
	}

	if !req.IsValidMethod() {
		logger.Warn().Str("method", req.Method).Msg("unsupported method. abort")

		return
	}

	domain := req.ExtractDomain()
	dstPort, err := req.ExtractPort()
	if err != nil {
		logger.Warn().Str("host", req.Host).Msg("failed to extract port")

		return
	}

	ctx = session.WithRemoteInfo(ctx, domain)
	logger = logger.With().Ctx(ctx).Logger()

	logger.Debug().
		Str("method", req.Method).
		Str("from", conn.RemoteAddr().String()).
		Msg("new request")

	nameMatch := p.ruleMatcher.Search(
		&matcher.Selector{Kind: matcher.MatchKindDomain, Domain: ptr.FromValue(domain)},
	)
	if nameMatch != nil && logger.GetLevel() == zerolog.TraceLevel {
		jsonAttrs, _ := json.Marshal(nameMatch)
		logger.Trace().RawJSON("values", jsonAttrs).Msg("name match")
	}

	t1 := time.Now()
	rSet, err := p.resolver.Resolve(ctx, domain, nil, nameMatch)
	dt := time.Since(t1).Milliseconds()
	if err != nil {
		_, _ = conn.Write(req.BadGatewayResponse())
		logging.ErrorUnwrapped(&logger, "dns lookup failed", err)

		return
	}

	logger.Debug().
		Int("cnt", len(rSet.Addrs)).
		Str("took", fmt.Sprintf("%dms", dt)).
		Msgf("dns lookup ok")

	// Avoid recursively querying self.
	if p.isRecursiveDst(ctx, rSet.Addrs, dstPort) {
		return
	}

	var selectors []*matcher.Selector
	for _, v := range rSet.Addrs {
		selectors = append(selectors, &matcher.Selector{
			Kind: matcher.MatchKindAddr,
			IP:   ptr.FromValue(v.IP),
			Port: ptr.FromValue(uint16(dstPort)),
		})
	}

	addrMatch := p.ruleMatcher.SearchAll(selectors)
	if addrMatch != nil && logger.GetLevel() == zerolog.TraceLevel {
		jsonAttrs, _ := json.Marshal(addrMatch)
		logger.Trace().RawJSON("values", jsonAttrs).Msg("addr match")
	}

	bestMatch := matcher.GetHigherPriorityRule(addrMatch, nameMatch)
	if bestMatch != nil && logger.GetLevel() == zerolog.TraceLevel {
		jsonAttrs, _ := json.Marshal(bestMatch)
		logger.Trace().RawJSON("values", jsonAttrs).Msg("best match")
	}

	if bestMatch != nil && *bestMatch.Block {
		logger.Debug().Msg("request is blocked by policy")
		return
	}

	var h handler.RequestHandler
	if req.IsConnectMethod() {
		h = p.httpsHandler
	} else {
		h = p.httpHandler
	}

	dst := &handler.Destination{
		Domain:  domain,
		Addrs:   rSet.Addrs,
		Port:    dstPort,
		Timeout: *p.serverOpts.Timeout,
	}

	handleErr := h.HandleRequest(ctx, conn, req, dst, bestMatch)
	if handleErr == nil { // Early exit if no error found
		return
	}

	logger.Warn().Err(handleErr).Msg("error handling request")
	if !errors.Is(handleErr, netutil.ErrBlocked) { // Early exit if not blocked
		return
	}

	// ┌─────────────┐
	// │ AUTO config │
	// └─────────────┘
	if nameMatch != nil {
		logger.Info().
			Str("match", *nameMatch.Match.Domain).
			Str("name", *nameMatch.Name).
			Msg("skipping auto-config (duplicate policy)")
		return
	}

	if addrMatch != nil {
		logger.Info().
			Str("match", addrMatch.Match.CIDR.String()).
			Str("name", *addrMatch.Name).
			Msg("skipping auto-config (duplicate policy)")
		return
	}

	// Perform auto config if enabled and RuleTemplate is not nil
	if *p.policyOpts.Auto && p.policyOpts.Template != nil {
		newRule := p.policyOpts.Template.Clone()
		newRule.Match = &config.MatchAttrs{Domain: ptr.FromValue(domain)}

		if err := p.ruleMatcher.Add(newRule); err != nil {
			logger.Info().Err(err).Msg("failed to add config automatically")
		} else {
			logger.Info().Msg("automatically added to config")
		}
	}
}

func (p *HTTPProxy) isRecursiveDst(
	ctx context.Context,
	dstAddrs []net.IPAddr,
	dstPort int,
) bool {
	logger := logging.WithLocalScope(ctx, p.logger, "is_recursive")

	if dstPort != int(p.serverOpts.ListenAddr.Port) {
		return false
	}

	for _, dstAddr := range dstAddrs {
		ip := dstAddr.IP
		if ip.IsLoopback() {
			logger.Trace().
				Str("addr", fmt.Sprintf("%s:%d", ip.String(), dstPort)).
				Msg("found a loopback destination")

			return true
		}

		// Get a list of available addresses.
		// See `ip -4 ifAddrs show`
		ifAddrs, err := net.InterfaceAddrs() // Needs AF_NETLINK on Linux.
		if err != nil {
			logger.Warn().Err(err).Msg("failed to retrieve interface addrs")
			return false
		}

		for _, addr := range ifAddrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ipnet.IP.Equal(ip) {
					logger.Debug().
						Str("addr", fmt.Sprintf("%s:%d", ip.String(), dstPort)).
						Msg("found a recursive destination")

					return true
				}
			}
		}
	}

	return false
}
