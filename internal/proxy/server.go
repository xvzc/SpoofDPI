package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/datastruct/tree"
	"github.com/xvzc/SpoofDPI/internal/dns"
	"github.com/xvzc/SpoofDPI/internal/logging"
	"github.com/xvzc/SpoofDPI/internal/packet"
	"github.com/xvzc/SpoofDPI/internal/proto"
	"github.com/xvzc/SpoofDPI/internal/session"
)

type ProxyOptions struct {
	ListenAddr    net.TCPAddr
	AutoPolicy    bool
	DNSQueryTypes []uint16
	Timeout       time.Duration
}

type Proxy struct {
	logger zerolog.Logger

	resolver     dns.Resolver
	httpHandler  Handler
	httpsHandler Handler
	domainTree   tree.SearchTree
	hopTracker   *packet.HopTracker
	opts         ProxyOptions
}

func NewProxy(
	logger zerolog.Logger,

	resolver dns.Resolver,
	httpHandler Handler,
	httpsHandler Handler,
	domainTree tree.SearchTree,
	hopTracker *packet.HopTracker,

	opts ProxyOptions,
) *Proxy {
	return &Proxy{
		logger:       logger,
		resolver:     resolver,
		httpHandler:  httpHandler,
		httpsHandler: httpsHandler,
		domainTree:   domainTree,
		hopTracker:   hopTracker,
		opts:         opts,
	}
}

func (pxy *Proxy) ListenAndServe(ctx context.Context, wait chan struct{}) {
	<-wait

	logger := pxy.logger.With().Ctx(ctx).Logger()

	listener, err := net.ListenTCP("tcp", &pxy.opts.ListenAddr)
	if err != nil {
		pxy.logger.Fatal().
			Err(err).
			Msgf("error creating listener on %s", pxy.opts.ListenAddr.String())
	}

	logger.Info().
		Msgf("created a listener on %s", pxy.opts.ListenAddr.String())

	for {
		conn, err := listener.Accept()
		if err != nil {
			pxy.logger.Error().
				Err(err).
				Msgf("failed to accept new connection")

			continue
		}

		go pxy.handleConnection(session.WithNewTraceID(context.Background()), conn)
	}
}

func (pxy *Proxy) handleConnection(ctx context.Context, conn net.Conn) {
	logger := logging.WithLocalScope(pxy.logger, ctx, "conn")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer closeConns(conn)

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
	port, err := req.ExtractPort()
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

	domainIncluded := pxy.checkDomainPolicy([]byte(domain))

	logger.Debug().Bool("include", domainIncluded).
		Msg("checked domain policy")

	ctx = session.WithPolicyIncluded(ctx, domainIncluded)

	t1 := time.Now()
	rSet, err := pxy.resolver.Resolve(ctx, domain, pxy.opts.DNSQueryTypes)
	dt := time.Since(t1).Milliseconds()
	if err != nil {
		_, _ = conn.Write(req.BadGatewayResponse())
		logging.ErrorUnwrapped(&logger, "dns lookup failed", err)

		return
	}

	logger.Debug().
		Int("len", rSet.Count()).
		Str("took", fmt.Sprintf("%dms", dt)).
		Msgf("dns lookup ok")

	dstAddrs := rSet.CopyAddrs()

	// Avoid recursively querying self.
	if pxy.isRecursiveDst(ctx, dstAddrs, port) {
		return
	}

	isPrivate := pxy.isPrivateDst(ctx, dstAddrs)
	shouldExploit := (!isPrivate && domainIncluded)
	ctx = session.WithShouldExploit(ctx, shouldExploit)

	if pxy.hopTracker != nil && shouldExploit {
		pxy.hopTracker.RegisterUntracked(dstAddrs, port)
	}

	var h Handler
	if req.IsConnectMethod() {
		h = pxy.httpsHandler
	} else {
		h = pxy.httpHandler
	}

	err = h.HandleRequest(ctx, conn, req, dstAddrs, port, pxy.opts.Timeout)
	if err == nil { // Early exit if no error found
		return
	}

	logger.Warn().Err(err).Msg("error handling request")
	if !errors.Is(err, errBlocked) { // Early exit if not blocked
		return
	}

	if pxy.opts.AutoPolicy && pxy.domainTree != nil { // Perform auto policy if enabled
		if added := pxy.addIncludedPolicy(domain); added {
			logger.Info().Msg("automatically added to policy")
		}
	}
}

func (pxy *Proxy) isRecursiveDst(
	ctx context.Context,
	dstAddrs []net.IPAddr,
	dstPort int,
) bool {
	logger := logging.WithLocalScope(pxy.logger, ctx, "is_recursive")

	if dstPort != int(pxy.opts.ListenAddr.Port) {
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
			logger.Trace().Err(err).Msg("failed to retrieve interface addrs")
			return false
		}

		for _, addr := range ifAddrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ipnet.IP.Equal(ip) {
					logger.Trace().
						Str("addr", fmt.Sprintf("%s:%d", ip.String(), dstPort)).
						Msg("found a recursive destination")

					return true
				}
			}
		}
	}

	return false
}

func (pxy *Proxy) isPrivateDst(ctx context.Context, dstAddrs []net.IPAddr) bool {
	logger := logging.WithLocalScope(pxy.logger, ctx, "is_private")

	for _, dstAddr := range dstAddrs {
		ip := dstAddr.IP
		if ip.IsPrivate() {
			logger.Trace().Str("ip", ip.String()).Msg("found a private ip addr")
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

	value, found := pxy.domainTree.Search(string(bytes))
	if found {
		return value.(bool)
	}

	return false
}

func (pxy *Proxy) addIncludedPolicy(domain string) bool {
	if _, found := pxy.domainTree.Search(domain); !found {
		pxy.domainTree.Insert(domain, true)
		return true
	}

	return false
}
