package proxy

import (
	"context"
	"io"
	"net"
	"os"
	"regexp"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/appctx"
	"github.com/xvzc/SpoofDPI/internal/dns"
)

type Proxy struct {
	logger zerolog.Logger

	resolver     dns.Resolver
	httpHandler  Handler
	httpsHandler Handler

	listenAddr      net.IP
	listenPort      uint16
	patternsAllowed []*regexp.Regexp
	patternsIgnored []*regexp.Regexp
	dnsQueryTypes   []uint16
	timeout         time.Duration
}

func NewProxy(
	logger zerolog.Logger,

	resolver dns.Resolver,
	httpHandler Handler,
	httpsHandler Handler,

	listenAddr net.IP,
	listenPort uint16,
	patternsAllowed []*regexp.Regexp,
	patternsIgnored []*regexp.Regexp,
	dnsQueryTypes []uint16,
	timeout time.Duration,
) *Proxy {
	return &Proxy{
		logger:          logger,
		resolver:        resolver,
		httpHandler:     httpHandler,
		httpsHandler:    httpsHandler,
		listenAddr:      listenAddr,
		listenPort:      listenPort,
		patternsAllowed: patternsAllowed,
		patternsIgnored: patternsIgnored,
		dnsQueryTypes:   dnsQueryTypes,
		timeout:         timeout,
	}
}

func (pxy *Proxy) ListenAndServe(wait chan struct{}) {
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
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger := pxy.logger.With().Ctx(ctx).Logger()

	req, err := readHttpRequest(conn)
	if err != nil {
		if err != io.EOF {
			logger.Warn().Msgf("error while parsing request: %s", err)
		}
		closeConns(conn)

		return
	}

	logger.Debug().
		Msgf("new request; from=%s; method=%s;", conn.RemoteAddr(), req.Method)

	if !req.IsValidMethod() {
		logger.Warn().Msgf("unsupported method; %s; abort;", req.Method)
		closeConns(conn)

		return
	}

	domain := req.ExtractDomain()
	port, err := req.ExtractPort()
	if err != nil {
		logger.Warn().Msgf("port extraction failed; host=%s; abort;", req.Host)
		closeConns(conn)
		return
	}

	patternMatched := patternMatches(
		[]byte(domain),
		pxy.patternsAllowed,
		pxy.patternsIgnored,
	)

	logger.Debug().Msgf("pattern matched; %v; %s;", patternMatched, domain)

	ctx = appctx.WithPatternMatched(ctx, patternMatched)

	t1 := time.Now()
	rSet, err := pxy.resolver.Resolve(ctx, domain, pxy.dnsQueryTypes)
	dt := time.Since(t1).Milliseconds()
	if err != nil || rSet.Counts() == 0 {
		logger.Warn().Msgf("dns result; name=%s; took=%dms; err=1;", domain, dt)
		logger.Warn().Msgf("error while dns lookup: %s", err)
		_, _ = conn.Write(req.ResBadGateway())
		closeConns(conn)
		return
	}
	logger.Debug().Msgf("dns result; name=%s; took=%dms; err=0;", domain, dt)

	dstAddrs := rSet.CopyAddrs()

	// Avoid recursively querying self.
	if pxy.isRecursive(ctx, dstAddrs, port) {
		logger.Error().Msg("looped request detected; abort;")
		closeConns(conn)
		return
	}

	var h Handler
	if req.IsConnectMethod() {
		h = pxy.httpsHandler
	} else {
		h = pxy.httpHandler
	}

	h.HandleRequest(ctx, conn, req, domain, dstAddrs, port, pxy.timeout)
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
