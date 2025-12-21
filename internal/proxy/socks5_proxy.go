package proxy

import (
	"context"
	"encoding/json"
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
	"github.com/xvzc/SpoofDPI/internal/proxy/handler"
	"github.com/xvzc/SpoofDPI/internal/ptr"
	"github.com/xvzc/SpoofDPI/internal/session"
)

type SOCKS5Proxy struct {
	logger zerolog.Logger

	resolver    dns.Resolver
	ruleMatcher matcher.RuleMatcher
	serverOpts  *config.ServerOptions
	policyOpts  *config.PolicyOptions

	tcpHandler *handler.TCPHandler
	udpHandler *handler.UDPHandler
}

func NewSOCKS5Proxy(
	logger zerolog.Logger,
	resolver dns.Resolver,
	bridge *handler.Bridge,
	ruleMatcher matcher.RuleMatcher,
	serverOpts *config.ServerOptions,
	policyOpts *config.PolicyOptions,
) ProxyServer {
	return &SOCKS5Proxy{
		logger:      logger,
		resolver:    resolver,
		ruleMatcher: ruleMatcher,
		serverOpts:  serverOpts,
		policyOpts:  policyOpts,
		tcpHandler: handler.NewTCPHandler(
			logger,
			bridge,
			serverOpts,
		),
		udpHandler: handler.NewUDPHandler(logger),
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

	// Setup Logging Context
	remoteInfo := req.Domain
	if remoteInfo == "" {
		remoteInfo = req.IP.String()
	}
	ctx = session.WithHostInfo(ctx, remoteInfo)

	switch req.Cmd {
	case proto.CmdConnect:
		rule, addrs, err := p.resolveAndMatch(ctx, req)
		if err != nil {
			return // resolveAndMatch logs error and writes failure response if needed
		}
		dst := &netutil.Destination{
			Domain:  req.Domain,
			Addrs:   addrs,
			Port:    req.Port,
			Timeout: *p.serverOpts.Timeout,
		}
		if err := p.tcpHandler.Handle(ctx, conn, req, dst, rule); err != nil {
			return // Handler logs error
		}

		p.handleAutoConfig(ctx, req, addrs, rule)

	case proto.CmdUDPAssociate:
		// UDP Associate usually doesn't have destination info in the request
		_ = p.udpHandler.Handle(ctx, conn, req, nil, nil)
	default:
		_ = proto.SOCKS5CommandNotSupportedResponse().Write(conn)
		logger.Warn().Uint8("cmd", req.Cmd).Msg("unsupported socks5 command")
	}
}

func (p *SOCKS5Proxy) resolveAndMatch(
	ctx context.Context,
	req *proto.SOCKS5Request,
) (*config.Rule, []net.IPAddr, error) {
	logger := zerolog.Ctx(ctx)

	// 1. Match Domain Rules (if domain provided)
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

	// 2. DNS Resolution
	t1 := time.Now()
	var addrs []net.IPAddr

	if req.Domain != "" {
		// Resolve Domain
		rSet, err := p.resolver.Resolve(ctx, req.Domain, nil, nameMatch)
		if err != nil {
			logging.ErrorUnwrapped(logger, "dns lookup failed", err)
			return nil, nil, err
		}
		addrs = rSet.Addrs
	} else {
		addrs = []net.IPAddr{{IP: req.IP}}
	}

	dt := time.Since(t1).Milliseconds()
	logger.Debug().
		Int("cnt", len(addrs)).
		Str("took", fmt.Sprintf("%dms", dt)).
		Msg("dns lookup ok")

	// 3. Match IP Rules
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
	return bestMatch, addrs, nil
}

func (p *SOCKS5Proxy) handleAutoConfig(
	ctx context.Context,
	req *proto.SOCKS5Request,
	addrs []net.IPAddr,
	matchedRule *config.Rule,
) {
	logger := zerolog.Ctx(ctx)

	if matchedRule != nil {
		logger.Info().
			Interface("match", matchedRule.Match).
			Str("name", *matchedRule.Name).
			Msg("skipping auto-config (duplicate policy)")
		return
	}

	if *p.policyOpts.Auto && p.policyOpts.Template != nil {
		newRule := p.policyOpts.Template.Clone()
		targetDomain := req.Domain
		if targetDomain == "" && len(addrs) > 0 {
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
