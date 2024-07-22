package proxy

import (
	"fmt"
	"os"
	"regexp"

	log "github.com/sirupsen/logrus"
	"github.com/xvzc/SpoofDPI/dns"
	"github.com/xvzc/SpoofDPI/net"
	"github.com/xvzc/SpoofDPI/packet"
	"github.com/xvzc/SpoofDPI/util"
)

type Proxy struct {
	addr           string
	port           int
	timeout        int
	resolver       *dns.DnsResolver
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
	}
}

func (pxy *Proxy) TcpAddr() *net.TCPAddr {
	return net.TcpAddr(pxy.addr, pxy.port)
}

func (pxy *Proxy) Port() int {
	return pxy.port
}

func (pxy *Proxy) Start() {
	l, err := net.ListenTCP("tcp4", pxy.TcpAddr())
	if err != nil {
		log.Fatal("Error creating listener: ", err)
		os.Exit(1)
	}

	log.Println(fmt.Sprintf("Connection timeout is set to %dms", pxy.timeout))

	log.Println("Created a listener on port", pxy.Port())

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal("Error accepting connection: ", err)
			continue
		}

		go func() {
			b, err := conn.ReadBytes()
			if err != nil {
				return
			}

			log.Debug("[PROXY] Request from ", conn.RemoteAddr(), "\n\n", string(b))

			pkt, err := packet.NewHttpPacket(b)
			if err != nil {
				log.Debug("Error while parsing request: ", string(b))
        conn.Close()
				return
			}

			if !pkt.IsValidMethod() {
				log.Debug("Unsupported method: ", pkt.Method())
        conn.Close()
				return
			}

			ip, err := pxy.resolver.Lookup(pkt.Domain())
			if err != nil {
				log.Error("[HTTP] Error looking up for domain with ", pkt.Domain(), " ", err)
				conn.Write([]byte(pkt.Version() + " 502 Bad Gateway\r\n\r\n"))
        conn.Close()
				return
			}

			if pkt.IsConnectMethod() {
				log.Debug("[HTTPS] Start")
				pxy.HandleHttps(conn, pkt, ip)
			} else {
				log.Debug("[HTTP] Start")
				pxy.HandleHttp(conn, pkt, ip)
			}
		}()
	}
}
