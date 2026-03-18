package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/rs/zerolog"
	"github.com/samber/lo"
	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/dns"
	"github.com/xvzc/SpoofDPI/internal/logging"
	"github.com/xvzc/SpoofDPI/internal/matcher"
	"github.com/xvzc/SpoofDPI/internal/netutil"
	"github.com/xvzc/SpoofDPI/internal/proto"
	"github.com/xvzc/SpoofDPI/internal/server"
	"github.com/xvzc/SpoofDPI/internal/session"
)

type HTTPProxy struct {
	logger zerolog.Logger

	resolver     dns.Resolver
	httpHandler  *HTTPHandler
	httpsHandler *HTTPSHandler
	ruleMatcher  matcher.RuleMatcher
	appOpts      *config.AppOptions
	connOpts     *config.ConnOptions
	policyOpts   *config.PolicyOptions

	listener net.Listener
}

func NewHTTPProxy(
	logger zerolog.Logger,
	resolver dns.Resolver,
	httpHandler *HTTPHandler,
	httpsHandler *HTTPSHandler,
	ruleMatcher matcher.RuleMatcher,
	appOpts *config.AppOptions,
	connOpts *config.ConnOptions,
	policyOpts *config.PolicyOptions,
) server.Server {
	return &HTTPProxy{
		logger:       logger,
		resolver:     resolver,
		httpHandler:  httpHandler,
		httpsHandler: httpsHandler,
		ruleMatcher:  ruleMatcher,
		appOpts:      appOpts,
		connOpts:     connOpts,
		policyOpts:   policyOpts,
	}
}

func (p *HTTPProxy) Start(ctx context.Context, ready chan<- struct{}) error {
	listener, err := net.ListenTCP("tcp", p.appOpts.ListenAddr)
	if err != nil {
		return fmt.Errorf(
			"error creating listener on %s: %w",
			p.appOpts.ListenAddr.String(),
			err,
		)
	}
	p.listener = listener

	if ready != nil {
		close(ready)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil // Normal shutdown
			}
			p.logger.Error().
				Err(err).
				Msgf("failed to accept new connection")

			continue
		}

		go p.handleNewConnection(session.WithNewTraceID(context.Background()), conn)
	}
}

func (p *HTTPProxy) Stop() error {
	if p.listener != nil {
		return p.listener.Close()
	}
	return nil
}

func (p *HTTPProxy) SetNetworkConfig() error {
	return SetSystemProxy(p.logger, uint16(p.appOpts.ListenAddr.Port))
}

func (p *HTTPProxy) UnsetNetworkConfig() error {
	return UnsetSystemProxy(p.logger)
}

func (p *HTTPProxy) Addr() string {
	return p.appOpts.ListenAddr.String()
}

func (p *HTTPProxy) handleNewConnection(ctx context.Context, conn net.Conn) {
	logger := logging.WithLocalScope(ctx, p.logger, "conn-init")

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

	logger.Debug().Str("from", conn.RemoteAddr().String()).Str("host", req.Host).
		Msg("new request")

	if !req.IsValidMethod() {
		logger.Warn().Str("method", req.Method).Msg("unsupported method. abort")
		_ = proto.HTTPNotImplementedResponse().Write(conn)

		return
	}

	host := req.ExtractHost()
	dstPort, err := req.ExtractPort()
	if err != nil {
		logger.Warn().Str("host", req.Host).Msg("failed to extract port")
		_ = proto.HTTPBadRequestResponse().Write(conn)

		return
	}

	logger.Debug().
		Str("method", req.Method).
		Str("from", conn.RemoteAddr().String()).
		Msg("new request")

	var addrs []net.IP
	var nameMatch *config.Rule
	if net.ParseIP(host) != nil {
		addrs = []net.IP{net.ParseIP(host)}
		logger.Trace().Msgf("skipping dns lookup for non-domain host %q", host)
	} else {
		nameMatch = p.ruleMatcher.Search(
			&matcher.Selector{Kind: matcher.MatchKindDomain, Domain: lo.ToPtr(host)},
		)

		rSet, err := p.resolver.Resolve(ctx, host, nil, nameMatch)
		if err != nil {
			_ = proto.HTTPBadGatewayResponse().Write(conn)
			// logging.ErrorUnwrapped is not available, using standard error logging
			logger.Error().Err(err).Msgf("dns lookup failed for %s", host)

			return
		}

		addrs = rSet.Addrs
	}

	// Avoid recursively querying self.
	ok, err := netutil.ValidateDestination(addrs, dstPort, p.appOpts.ListenAddr)
	if err != nil {
		logger.Debug().Err(err).Msg("error validating dst addrs")
		if !ok {
			_ = proto.HTTPForbiddenResponse().Write(conn)
		}
	}

	var selectors []*matcher.Selector
	for _, v := range addrs {
		selectors = append(selectors, &matcher.Selector{
			Kind: matcher.MatchKindAddr,
			IP:   lo.ToPtr(v),
			Port: lo.ToPtr(uint16(dstPort)),
		})
	}

	addrMatch := p.ruleMatcher.SearchAll(selectors)

	bestMatch := matcher.GetHigherPriorityRule(addrMatch, nameMatch)
	if bestMatch != nil && logger.GetLevel() == zerolog.TraceLevel {
		logger.Trace().RawJSON("summary", bestMatch.JSON()).Msg("match")
	}

	if bestMatch != nil && *bestMatch.Block {
		logger.Debug().Msg("request is blocked by policy")
		return
	}

	dst := &netutil.Destination{
		Domain:  host, // Updated from Domain to Host
		Addrs:   addrs,
		Port:    dstPort,
		Timeout: *p.connOpts.TCPTimeout,
	}

	var handleErr error
	if req.IsConnectMethod() {
		handleErr = p.httpsHandler.HandleRequest(ctx, conn, dst, bestMatch)
	} else {
		handleErr = p.httpHandler.HandleRequest(ctx, conn, req, dst, bestMatch)
	}

	if handleErr == nil { // Early exit if no error found
		return
	}

	logger.Warn().Err(handleErr).Msg("error handling request")
}
