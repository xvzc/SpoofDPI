package proxy

import (
	"log"
	"os"

	"github.com/xvzc/SpoofDPI/doh"
	"github.com/xvzc/SpoofDPI/net"
	"github.com/xvzc/SpoofDPI/packet"
)

type Proxy struct {
	Port string
}

func New(port string) *Proxy {
	return &Proxy{
		Port: port,
	}
}

func (p *Proxy) Start() {
	listener, err := net.Listen("tcp", ":"+p.Port)
	if err != nil {
		log.Fatal("Error creating listener: ", err)
		os.Exit(1)
	}

	// util.Debug("Created a listener")

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			log.Fatal("Error accepting connection: ", err)
			continue
		}

		// util.Debug("Accepted a new connection.", clientConn.RemoteAddr())

		go func() {
			defer clientConn.Close()

			b, err := clientConn.ReadBytes()
			if err != nil {
				return
			}

			// util.Debug("Client sent data: ", len(b))

			r := packet.NewHttpPacket(&b)
			// util.Debug("Request: \n" + string(*r.Raw))

			if !r.IsValidMethod() {
				log.Println("Unsupported method: ", r.Method)
				return
			}

			// Dns lookup over https
			ip, err := doh.Lookup(r.Domain)
			if err != nil {
				log.Println("Error looking up dns: "+r.Domain, err)
				return
			}

			// util.Debug("ip: " + ip)

			if r.IsConnectMethod() {
				// util.Debug("HTTPS Requested")
				HandleHttps(clientConn, ip, &r)
			} else {
				// util.Debug("HTTP Requested.")
				HandleHttp(clientConn, ip, &r)
			}
		}()
	}
}
