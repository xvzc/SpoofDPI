package proxy

import (
	"context"
	"net"
	"os"
	"regexp"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/appctx"
	"github.com/xvzc/SpoofDPI/internal/dns"
)

type Proxy struct {
	listenAddr      net.IP
	listenPort      uint16
	patternsAllowed []*regexp.Regexp
	patternsIgnored []*regexp.Regexp
	timeout         time.Duration

	httpHandler  Handler
	httpsHandler Handler
	resolver     dns.Resolver
	logger       zerolog.Logger
}

func NewProxy(
	listenAddr net.IP,
	listenPort uint16,
	patternsAllowed []*regexp.Regexp,
	patternsIgnored []*regexp.Regexp,
	timeout time.Duration,
	resolver dns.Resolver,
	httpHandler Handler,
	httpsHandler Handler,
	logger zerolog.Logger,
) *Proxy {
	return &Proxy{
		listenAddr:      listenAddr,
		listenPort:      listenPort,
		timeout:         timeout,
		patternsAllowed: patternsAllowed,
		patternsIgnored: patternsIgnored,

		resolver:     resolver,
		httpHandler:  httpHandler,
		httpsHandler: httpsHandler,
		logger:       logger,
	}
}

func (pxy *Proxy) Start(wait chan struct{}) {
	<-wait

	logger := pxy.logger

	listener, err := net.ListenTCP(
		"tcp",
		&net.TCPAddr{IP: pxy.listenAddr, Port: int(pxy.listenPort)},
	)
	if err != nil {
		pxy.logger.Fatal().Msgf("error creating listener: %s", err)
		os.Exit(1)
	}

	logger.Info().Msgf("created a listener(%s:%d)", pxy.listenAddr, pxy.listenPort)

	for {
		conn, err := listener.Accept()
		if err != nil {
			pxy.logger.Fatal().Msgf("error accepting connection: %s", err)

			continue
		}

		go pxy.handleConnection(appctx.WithNewTraceID(context.Background()), conn)
	}
}

func (pxy *Proxy) handleConnection(ctx context.Context, conn net.Conn) {
	ctx, cancel := context.WithCancel(appctx.WithNewTraceID(ctx))
	defer cancel()

	logger := pxy.logger.With().Ctx(ctx).Logger()

	req, err := readHttpRequest(conn)
	if err != nil {
		logger.Debug().Msgf("error while parsing request: %s", err)
		closeConns(conn)

		return
	}

	logger.Debug().
		Msgf("new request from %s, method: %s", conn.RemoteAddr(), req.Method)

	if !req.IsValidMethod() {
		logger.Debug().Msgf("unsupported method: %s", req.Method)
		closeConns(conn)

		return
	}

	domain := req.ExtractDomain()
	port, err := req.ExtractPort()
	if err != nil {
		logger.Debug().Msgf(
			"error while extracting port from request %s: %s",
			req.Host,
			err,
		)
		closeConns(conn)
		return
	}

	patternMatched := patternMatches(
		[]byte(domain),
		pxy.patternsAllowed,
		pxy.patternsIgnored,
	)

	ctx = appctx.WithPatternMatched(ctx, patternMatched)

	rSet, err := pxy.resolver.Resolve(ctx, domain)
	if err != nil {
		logger.Debug().Msgf("error while dns lookup: %s %s", domain, err)
		_, _ = conn.Write([]byte(req.Proto + " 502 Bad Gateway\r\n\r\n"))
		closeConns(conn)
		return
	}

	dstAddrs := rSet.CopyAddrs()

	// Avoid recursively querying self.
	if pxy.isRecursive(ctx, dstAddrs, port) {
		logger.Error().Msg("detected a looped request. aborting.")
		closeConns(conn)
		return
	}

	var h Handler
	if req.IsConnectMethod() {
		h = pxy.httpsHandler
	} else {
		h = pxy.httpHandler
	}

	h.Serve(ctx, conn, req, domain, dstAddrs, port, pxy.timeout)
}

func (pxy *Proxy) isRecursive(
	ctx context.Context,
	dstAddrs []net.IPAddr,
	dstPort int,
) bool {
	logger := pxy.logger.With().Ctx(ctx).Logger()

	if dstPort != int(pxy.listenPort) {
		return false
	}

	for _, dstAddr := range dstAddrs {
		ip := net.ParseIP(dstAddr.String())
		if ip.IsLoopback() {
			return true
		}

		// Get a list of available addresses.
		// See `ip -4 ifAddrs show`
		ifAddrs, err := net.InterfaceAddrs() // Needs AF_NETLINK on Linux.
		if err != nil {
			logger.Error().Msgf("error retrieving addrs of network interfaces: %s", err)
			return false
		}

		for _, addr := range ifAddrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ipnet.IP.Equal(ip) {
					return true
				}
			}
		}
	}

	return false
}

func patternMatches(
	bytes []byte,
	allow []*regexp.Regexp,
	ignore []*regexp.Regexp,
) bool {
	// always true when there's no patterns to check
	if len(allow) == 0 && len(ignore) == 0 {
		return true
	}

	// use whitelist strategy when allow patterns exist
	// skip checking for ignore patterns this case
	if len(allow) > 0 {
		for _, p := range allow {
			if p.Match(bytes) {
				return true
			}
		}

		return false
	}

	// use blacklist strategy when only ignore patterns exist
	for _, p := range ignore {
		if p.Match(bytes) {
			return false
		}
	}

	return true
}
