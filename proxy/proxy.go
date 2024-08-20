package proxy

import (
	"fmt"
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
	addr              string
	port              int
	timeout           int
	resolver          *dns.DnsResolver
	windowSize        int
	allowedPattern    []*regexp.Regexp
	disallowedPattern []*regexp.Regexp
}

func New(config *util.Config) *Proxy {
	return &Proxy{
		addr:              *config.Addr,
		port:              *config.Port,
		timeout:           *config.Timeout,
		windowSize:        *config.WindowSize,
		allowedPattern:    config.AllowedPatterns,
		disallowedPattern: config.DisallowedPatterns,
		resolver:          dns.NewResolver(config),
	}
}

func (pxy *Proxy) Start() {
	l, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.ParseIP(pxy.addr), Port: pxy.port})
	if err != nil {
		log.Fatal("[PROXY] error creating listener: ", err)
		os.Exit(1)
	}

	if pxy.timeout > 0 {
		log.Println(fmt.Sprintf("[PROXY] connection timeout is set to %dms", pxy.timeout))
	}

	log.Println("[PROXY] created a listener on port", pxy.port)
	if len(pxy.allowedPattern) > 0 {
		log.Println("[PROXY] number of white-listed pattern:", len(pxy.allowedPattern))
	}
	if len(pxy.disallowedPattern) > 0 {
		log.Println("[PROXY] number of black-listed pattern:", len(pxy.disallowedPattern))
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal("[PROXY] error accepting connection: ", err)
			continue
		}

		go func() {
			pkt, err := packet.ReadHttpPacket(conn)
			if err != nil {
				log.Debug("[PROXY] error while parsing request: ", err)
				conn.Close()
				return
			}

			log.Debug("[PROXY] request from ", conn.RemoteAddr(), "\n\n", string(pkt.Raw()))

			if !pkt.IsValidMethod() {
				log.Debug("[PROXY] unsupported method: ", pkt.Method())
				conn.Close()
				return
			}

			matched := pxy.patternMatches([]byte(pkt.Domain()))
			useSystemDns := !matched

			ip, err := pxy.resolver.Lookup(pkt.Domain(), useSystemDns)
			if err != nil {
				log.Debug("[PROXY] error while dns lookup: ", pkt.Domain(), " ", err)
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
	if pxy.disallowedPattern != nil {
		for _, pattern := range pxy.disallowedPattern {
			if pattern.Match(bytes) {
				return false
			}
		}
	}

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
		log.Error("[PROXY] error while getting addresses of our network interfaces: ", err)
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
