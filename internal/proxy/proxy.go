package proxy

import (
	"context"
	"io"
	"net"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/appctx"
	"github.com/xvzc/SpoofDPI/internal/datastruct/tree"
	"github.com/xvzc/SpoofDPI/internal/dns"
)

type Proxy struct {
	logger zerolog.Logger

	resolver     dns.Resolver
	httpHandler  Handler
	httpsHandler Handler
	domainTree   tree.RadixTree

	listenAddr    net.IP
	listenPort    uint16
	dnsQueryTypes []uint16
	timeout       time.Duration
}

func NewProxy(
	logger zerolog.Logger,

	resolver dns.Resolver,
	httpHandler Handler,
	httpsHandler Handler,
	domainTree tree.RadixTree,

	listenAddr net.IP,
	listenPort uint16,
	dnsQueryTypes []uint16,
	timeout time.Duration,
) *Proxy {
	return &Proxy{
		logger:        logger,
		resolver:      resolver,
		httpHandler:   httpHandler,
		httpsHandler:  httpsHandler,
		domainTree:    domainTree,
		listenAddr:    listenAddr,
		listenPort:    listenPort,
		dnsQueryTypes: dnsQueryTypes,
		timeout:       timeout,
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

	domainIncluded := pxy.checkDomainPolicy([]byte(domain))

	logger.Debug().Msgf("domain policy; name=%s; include=%v;", domain, domainIncluded)

	ctx = appctx.WithDomainIncluded(ctx, domainIncluded)

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
	if pxy.isRecursiveDst(ctx, dstAddrs, port) {
		closeConns(conn)

		return
	}

	isPrivate := pxy.isPrivateDst(ctx, dstAddrs)
	ctx = appctx.WithShouldExploit(ctx, (!isPrivate && domainIncluded))

	var h Handler
	if req.IsConnectMethod() {
		h = pxy.httpsHandler
	} else {
		h = pxy.httpHandler
	}

	h.HandleRequest(ctx, conn, req, domain, dstAddrs, port, pxy.timeout)
}

func (pxy *Proxy) isRecursiveDst(
	ctx context.Context,
	dstAddrs []net.IPAddr,
	dstPort int,
) bool {
	logger := pxy.logger.With().Ctx(ctx).Logger()

	if dstPort != int(pxy.listenPort) {
		return false
	}

	for _, dstAddr := range dstAddrs {
		ip := dstAddr.IP
		if ip.IsLoopback() {
			logger.Trace().Msgf("recursive dst; ip=%s; port=%d; abort;",
				ip.String(), dstPort,
			)
			return true
		}

		// Get a list of available addresses.
		// See `ip -4 ifAddrs show`
		ifAddrs, err := net.InterfaceAddrs() // Needs AF_NETLINK on Linux.
		if err != nil {
			logger.Error().
				Msgf("recursive dst; error retrieving addrs of network interfaces: %s", err)
			return false
		}

		for _, addr := range ifAddrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ipnet.IP.Equal(ip) {
					logger.Trace().Msgf("recursive dst; ip=%s; port=%d; abort;",
						ip.String(), dstPort,
					)
					return true
				}
			}
		}
	}

	return false
}

func (pxy *Proxy) isPrivateDst(ctx context.Context, dstAddrs []net.IPAddr) bool {
	logger := pxy.logger.With().Ctx(ctx).Logger()

	for _, dstAddr := range dstAddrs {
		ip := dstAddr.IP
		if ip.IsPrivate() {
			logger.Trace().Msgf("private dst; ip=%s;", ip.String())
			return true
		}
	}

	return false
}

func (pxy *Proxy) checkDomainPolicy(
	bytes []byte,
) bool {
	// always return true when there's no patterns to check
	if pxy.domainTree == nil {
		return true
	}

	value, found := pxy.domainTree.Lookup(string(bytes))
	if found {
		return value.(bool)
	}

	return false
}
