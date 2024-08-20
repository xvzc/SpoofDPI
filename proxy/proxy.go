package proxy

import (
	"net"
	"os"
	"regexp"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/xvzc/SpoofDPI/dns"
	"github.com/xvzc/SpoofDPI/packet"
	"github.com/xvzc/SpoofDPI/util"
)

type Proxy struct {
	addr           string
	port           int
	timeout        int
	resolver       *dns.DnsResolver
	windowSize     int
	allowedPattern []*regexp.Regexp
}

func New(config *util.Config) *Proxy {
	return &Proxy{
		addr:           *config.Addr,
		port:           *config.Port,
		timeout:        *config.Timeout,
		windowSize:     *config.WindowSize,
		allowedPattern: config.AllowedPatterns,
		resolver:       dns.NewResolver(config),
	}
}

func (pxy *Proxy) Start() {
	l, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.ParseIP(pxy.addr), Port: pxy.port})
	if err != nil {
		log.Fatalf("[PROXY] error creating listener: %s", err)
		os.Exit(1)
	}

	if pxy.timeout > 0 {
		log.Infof("[PROXY] connection timeout is set to %dms", pxy.timeout)
	}

	log.Infof("[PROXY] created a listener on port %d", pxy.port)
	if len(pxy.allowedPattern) > 0 {
		log.Infof("[PROXY] number of white-listed pattern: %d", len(pxy.allowedPattern))
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatalf("[PROXY] error accepting connection: %s", err)
			continue
		}

		go func() {
			pkt, err := packet.ReadHttpPacket(conn)
			if err != nil {
				log.Debugf("[PROXY] error while parsing request: %s", err)
				conn.Close()
				return
			}

			log.Debugf("[PROXY] request from %s\n\n%s", conn.RemoteAddr(), string(pkt.Raw()))

			if !pkt.IsValidMethod() {
				log.Debugf("[PROXY] unsupported method: %s", pkt.Method())
				conn.Close()
				return
			}

			matched := pxy.patternMatches([]byte(pkt.Domain()))
			useSystemDns := !matched

			ip, err := pxy.resolver.Lookup(pkt.Domain(), useSystemDns)
			if err != nil {
				log.Debugf("[PROXY] error while resolving domain name: %s: %s", pkt.Domain(), err)
				conn.Write([]byte(pkt.Version() + " 502 Bad Gateway\r\n\r\n"))
				conn.Close()
				return
			}

			// Avoid recursively querying self
			if pkt.Port() == strconv.Itoa(pxy.port) && isLoopedRequest(net.ParseIP(ip)) {
				log.Error("[PROXY] looped request has been detected. aborting.")
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
	// we don't handle IPv6 at all it seems
	if ip.To4() == nil {
		return false
	}

	if ip.IsLoopback() {
		return true
	}

	// Get list of available addresses
	// See `ip -4 addr show`
	addr, err := net.InterfaceAddrs() // needs AF_NETLINK on linux
	if err != nil {
		log.Errorf("[PROXY] error while getting addresses of local network interfaces: %s", err)
		return false
	}

	for _, addr := range addr {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if ipnet.IP.To4() != nil && ipnet.IP.To4().Equal(ip) {
				return true
			}
		}
	}

	return false
}
