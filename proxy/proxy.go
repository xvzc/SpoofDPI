package proxy

import (
	"context"
	"fmt"
	"math/rand"
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
	systemResolver *net.Resolver
	windowSize     int
	allowedPattern *regexp.Regexp
	allowedUrls    *regexp.Regexp
}

func New(config *util.Config) *Proxy {
	return &Proxy{
		addr:           *config.Addr,
		port:           *config.Port,
		timeout:        *config.Timeout,
		windowSize:     *config.WindowSize,
		allowedPattern: config.AllowedPattern,
		allowedUrls:    config.AllowedUrls,
		resolver:       dns.NewResolver(config),
		systemResolver: &net.Resolver{PreferGo: true},
	}
}

func (pxy *Proxy) Start() {
	l, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.ParseIP(pxy.addr), Port: pxy.port})
	if err != nil {
		log.Fatal("[PROXY] Error creating listener: ", err)
		os.Exit(1)
	}

	if pxy.timeout > 0 {
		log.Println(fmt.Sprintf("[PROXY] Connection timeout is set to %dms", pxy.timeout))
	}

	log.Println("[PROXY] Created a listener on port", pxy.port)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal("[PROXY] Error accepting connection: ", err)
			continue
		}

		go func() {
			b, err := ReadBytes(conn.(*net.TCPConn))
			if err != nil {
				return
			}

			log.Debug("[PROXY] Request from ", conn.RemoteAddr(), "\n\n", string(b))

			pkt, err := packet.NewHttpPacket(b)
			if err != nil {
				log.Debug("[PROXY] Error while parsing request: ", string(b))
				conn.Close()
				return
			}

			if !pkt.IsValidMethod() {
				log.Debug("[PROXY] Unsupported method: ", pkt.Method())
				conn.Close()
				return
			}

			var ip string
			if pxy.patternExists() && !pxy.patternMatches([]byte(pkt.Domain())) {
				ips, err := pxy.systemResolver.LookupIPAddr(context.Background(), pkt.Domain())
				if err != nil {
					log.Error("[PROXY] Error while dns lookup: ", pkt.Domain(), " ", err)
					conn.Write([]byte(pkt.Version() + " 502 Bad Gateway\r\n\r\n"))
					conn.Close()
					return
				}

				if len(ips) == 0 {
					log.Error("[PROXY] Error while dns lookup: ", pkt.Domain(), " len(ips) = ", len(ips))
					conn.Write([]byte(pkt.Version() + " 502 Bad Gateway\r\n\r\n"))
					conn.Close()
					return
				}

				ip = ips[rand.Intn(len(ips))].String()
			} else {
				ip, err = pxy.resolver.Lookup(pkt.Domain())
				if err != nil {
					log.Error("[PROXY] Error while dns lookup: ", pkt.Domain(), " ", err)
					conn.Write([]byte(pkt.Version() + " 502 Bad Gateway\r\n\r\n"))
					conn.Close()
					return
				}
			}

			// Avoid recursively querying self
			if pkt.Port() == strconv.Itoa(pxy.port) && isLoopedRequest(net.ParseIP(ip)) {
				log.Error("[PROXY] Looped request has been detected. aborting.")
				conn.Close()
				return
			}

			if pkt.IsConnectMethod() {
				log.Debug("[PROXY] Start HTTPS")
				pxy.handleHttps(conn.(*net.TCPConn), pkt, ip)
			} else {
				log.Debug("[PROXY] Start HTTP")
				pxy.handleHttp(conn.(*net.TCPConn), pkt, ip)
			}
		}()
	}
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
		log.Error("[PROXY] Error while getting addresses of our network interfaces: ", err)
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
