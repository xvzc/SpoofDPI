package proxy

import (
	"net"
	"os"
	"regexp"
	"strconv"

	"github.com/xvzc/SpoofDPI/dns"
	"github.com/xvzc/SpoofDPI/packet"
	"github.com/xvzc/SpoofDPI/util"
	"github.com/xvzc/SpoofDPI/util/log"
)

const scopeProxy = "PROXY"

type Proxy struct {
	addr           string
	port           int
	timeout        int
	resolver       *dns.Dns
	windowSize     int
	enableDoh      bool
	allowedPattern []*regexp.Regexp
}

func New(config *util.Config) *Proxy {
	return &Proxy{
		addr:           *config.Addr,
		port:           *config.Port,
		timeout:        *config.Timeout,
		windowSize:     *config.WindowSize,
		enableDoh:      *config.EnableDoh,
		allowedPattern: config.AllowedPatterns,
		resolver:       dns.NewDns(config),
	}
}

func (pxy *Proxy) Start() {
	l, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP(pxy.addr), Port: pxy.port})
	if err != nil {
		log.Logger.Fatal().
			Str(log.ScopeFieldName, scopeProxy).
			Msgf("error creating listener: %s", err)
		os.Exit(1)
	}

	if pxy.timeout > 0 {
		log.Logger.Debug().
			Str(log.ScopeFieldName, scopeProxy).
			Msgf("connection timeout is set to %d ms", pxy.timeout)
	}

	log.Logger.Info().Msgf("created a listener on port %d", pxy.port)
	if len(pxy.allowedPattern) > 0 {
		log.Logger.Debug().
			Str(log.ScopeFieldName, scopeProxy).
			Msgf("number of white-listed pattern: %d", len(pxy.allowedPattern))
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Logger.Fatal().
				Str(log.ScopeFieldName, scopeProxy).
				Msgf("error accepting connection: %s", err)
			continue
		}

		go func() {
			pkt, err := packet.ReadHttpRequest(conn)
			if err != nil {
				log.Logger.Debug().
					Str(log.ScopeFieldName, scopeProxy).
					Msgf("error while parsing request: %s", err)
				conn.Close()
				return
			}

			log.Logger.Debug().
				Str(log.ScopeFieldName, scopeProxy).
				Msgf("request from %s\n\n%s", conn.RemoteAddr(), pkt.Raw())

			if !pkt.IsValidMethod() {
				log.Logger.Debug().
					Str(log.ScopeFieldName, scopeProxy).
					Msgf("unsupported method: %s", pkt.Method())
				conn.Close()
				return
			}

			matched := pxy.patternMatches([]byte(pkt.Domain()))
			useSystemDns := !matched

			ip, err := pxy.resolver.ResolveHost(pkt.Domain(), pxy.enableDoh, useSystemDns)
			if err != nil {
				log.Logger.Debug().
					Str(log.ScopeFieldName, scopeProxy).
					Msgf("error while dns lookup: %s %s", pkt.Domain(), err)
				conn.Write([]byte(pkt.Version() + " 502 Bad Gateway\r\n\r\n"))
				conn.Close()
				return
			}

			// Avoid recursively querying self
			if pkt.Port() == strconv.Itoa(pxy.port) && isLoopedRequest(net.ParseIP(ip)) {
				log.Logger.Error().
					Str(log.ScopeFieldName, scopeProxy).
					Msg("looped request has been detected. aborting.")
				conn.Close()
				return
			}

			if pkt.IsConnectMethod() {
				pxy.handleHttps(conn.(*net.TCPConn), matched, pkt, ip)
			} else {
				pxy.handleHttp(conn.(*net.TCPConn), pkt, ip)
			}
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

func isLoopedRequest(ip net.IP) bool {
	if ip.IsLoopback() {
		return true
	}

	// Get list of available addresses
	// See `ip -4 addr show`
	addr, err := net.InterfaceAddrs() // needs AF_NETLINK on linux
	if err != nil {
		log.Logger.Error().
			Str(log.ScopeFieldName, scopeProxy).
			Msgf("error while getting addresses of our network interfaces: %s", err)
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
