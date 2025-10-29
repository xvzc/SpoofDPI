package proxy

import (
	"context"
	"net"
	"os"
	"regexp"
	"strconv"

	"github.com/xvzc/SpoofDPI/config"
	"github.com/xvzc/SpoofDPI/dns"
	"github.com/xvzc/SpoofDPI/packet"
	"github.com/xvzc/SpoofDPI/proxy/handler"
	"github.com/xvzc/SpoofDPI/proxy/handler/http"
	"github.com/xvzc/SpoofDPI/proxy/handler/https"
	"github.com/xvzc/SpoofDPI/util"
	"github.com/xvzc/SpoofDPI/util/log"
)

const scopeProxy = "PROXY"

type Proxy struct {
	resolver *dns.Dns
}

func New() *Proxy {
	return &Proxy{
		resolver: dns.NewDns(),
	}
}

func (pxy *Proxy) Start(ctx context.Context) {
	c := config.Get()

	ctx = util.GetCtxWithScope(ctx, scopeProxy)
	logger := log.GetCtxLogger(ctx)

	l, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.ParseIP(c.Addr()),
		Port: c.Port(),
	})

	if err != nil {
		logger.Fatal().Msgf("error creating listener: %s", err)
		os.Exit(1)
	}

	if c.Timeout() > 0 {
		logger.Info().Msgf("connection timeout is set to %d ms", c.Timeout())
	}

	logger.Info().Msgf("created a listener on port %d", c.Port())
	if len(c.AllowedPatterns()) > 0 {
		logger.Info().Msgf("number of white-listed pattern: %d", len(c.AllowedPatterns()))
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

			matched := pxy.patternMatches([]byte(pkt.Domain()), c.AllowedPatterns())
			useSystemDns := !matched
			ctx = context.WithValue(ctx, "patternMatched", matched)
			ctx = context.WithValue(ctx, "shouldExploit", matched)
			ctx = context.WithValue(ctx, "useSystemDns", useSystemDns)

			ip, err := pxy.resolver.ResolveHost(ctx, pkt.Domain())
			if err != nil {
				logger.Debug().Msgf("error while dns lookup: %s %s", pkt.Domain(), err)
				conn.Write([]byte(pkt.Version() + " 502 Bad Gateway\r\n\r\n"))
				conn.Close()
				return
			}

			// Avoid recursively querying self
			if pkt.Port() == strconv.Itoa(c.Port()) && isLoopedRequest(ctx, net.ParseIP(ip)) {
				logger.Error().Msg("looped request has been detected. aborting.")
				conn.Close()
				return
			}

			var h handler.Handler
			if pkt.IsConnectMethod() {
				h = https.GetInstance()
			} else {
				h = http.GetInstance()
			}

			h.Serve(ctx, conn.(*net.TCPConn), pkt, ip)
		}()
	}
}

func (pxy *Proxy) patternMatches(bytes []byte, patterns []*regexp.Regexp) bool {

	if patterns == nil {
		return true
	}

	for _, pattern := range patterns {
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
