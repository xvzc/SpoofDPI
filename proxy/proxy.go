package proxy

import (
	"context"
	"net"
	"os"
	"regexp"
	"strconv"

	"github.com/xvzc/SpoofDPI/dns"
	"github.com/xvzc/SpoofDPI/packet"
	"github.com/xvzc/SpoofDPI/proxy/handler"
	"github.com/xvzc/SpoofDPI/util"
	"github.com/xvzc/SpoofDPI/util/log"
)

const scopeProxy = "PROXY"

type Proxy struct {
	addr           string
	port           int
	timeout        int
	resolver       *dns.Dns
	disorder       bool
	windowSize     int
	enableDoh      bool
	allowedPattern []*regexp.Regexp
}

type Handler interface {
	Serve(ctx context.Context, lConn *net.TCPConn, pkt *packet.HttpRequest, ip string)
}

func New(config *util.Config) *Proxy {
	return &Proxy{
		addr:           config.Addr,
		port:           config.Port,
		timeout:        config.Timeout,
		disorder:       config.Disorder,
		windowSize:     config.WindowSize,
		enableDoh:      config.EnableDoh,
		allowedPattern: config.AllowedPatterns,
		resolver:       dns.NewDns(config),
	}
}

func (pxy *Proxy) Start(ctx context.Context) {
	ctx = util.GetCtxWithScope(ctx, scopeProxy)
	logger := log.GetCtxLogger(ctx)

	l, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP(pxy.addr), Port: pxy.port})
	if err != nil {
		logger.Fatal().Msgf("error creating listener: %s", err)
		os.Exit(1)
	}

	if pxy.timeout > 0 {
		logger.Info().Msgf("connection timeout is set to %d ms", pxy.timeout)
	}

	logger.Info().Msgf("created a listener on port %d", pxy.port)
	if len(pxy.allowedPattern) > 0 {
		logger.Info().Msgf("number of white-listed pattern: %d", len(pxy.allowedPattern))
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			logger.Fatal().Msgf("error accepting connection: %s", err)
			continue
		}

		go func() {
			ctx := util.GetCtxWithTraceId(ctx)
			logger := log.GetCtxLogger(ctx)

			pkt, err := packet.ReadHttpRequest(conn)
			if err != nil {
				logger.Debug().Msgf("error while parsing request: %s", err)
				conn.Close()
				return
			}

			pkt.Tidy()

			logger.Debug().Msgf("request from %s\n\n%s", conn.RemoteAddr(), string(pkt.Raw()))

			if !pkt.IsValidMethod() {
				logger.Debug().Msgf("unsupported method: %s", pkt.Method())
				conn.Close()
				return
			}

			matched := pxy.patternMatches([]byte(pkt.Domain()))
			useSystemDns := !matched

			ip, err := pxy.resolver.ResolveHost(ctx, pkt.Domain(), pxy.enableDoh, useSystemDns)
			if err != nil {
				logger.Debug().Msgf("error while dns lookup: %s %s", pkt.Domain(), err)
				conn.Write([]byte(pkt.Version() + " 502 Bad Gateway\r\n\r\n"))
				conn.Close()
				return
			}

			// Avoid recursively querying self
			if pkt.Port() == strconv.Itoa(pxy.port) && isLoopedRequest(ctx, net.ParseIP(ip)) {
				logger.Error().Msg("looped request has been detected. aborting.")
				conn.Close()
				return
			}

			var h Handler
			if pkt.IsConnectMethod() {
				h = handler.NewHttpsHandler(pxy.timeout, pxy.windowSize, pxy.allowedPattern, matched, pxy.disorder)
			} else {
				h = handler.NewHttpHandler(pxy.timeout)
			}

			h.Serve(ctx, conn.(*net.TCPConn), pkt, ip)
		}()
	}
}

func (pxy *Proxy) patternMatches(bytes []byte) bool {
	if pxy.allowedPattern == nil {
		return true
	}

	for _, pattern := range pxy.allowedPattern {
		if pattern.Match(bytes) {
			return true
		}
	}

	return false
}

func isLoopedRequest(ctx context.Context, ip net.IP) bool {
	if ip.IsLoopback() {
		return true
	}

	logger := log.GetCtxLogger(ctx)

	// Get list of available addresses
	// See `ip -4 addr show`
	addr, err := net.InterfaceAddrs() // needs AF_NETLINK on linux
	if err != nil {
		logger.Error().Msgf("error while getting addresses of our network interfaces: %s", err)
		return false
	}

	for _, addr := range addr {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if ipnet.IP.Equal(ip) {
				return true
			}
		}
	}

	return false
}
