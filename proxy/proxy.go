package proxy

import (
	"os"

	log "github.com/sirupsen/logrus"
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
	l, err := net.Listen("tcp", ":"+p.Port)
	if err != nil {
		log.Fatal("Error creating listener: ", err)
		os.Exit(1)
	}

	log.Println("Created a listener on :", p.Port)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal("Error accepting connection: ", err)
			continue
		}

		log.Debug("Accepted a new connection.", conn.RemoteAddr())

		go func() {
			defer conn.Close()

			b, err := conn.ReadBytes()
			if err != nil {
				return
			}
			log.Debug("Client sent data: ", len(b))

			r := packet.NewHttpPacket(b)
			log.Debug("New request: \n\n" + string(r.Raw))

			if !r.IsValidMethod() {
				log.Println("Unsupported method: ", r.Method)
				return
			}

			if r.IsConnectMethod() {
				log.Debug("HTTPS Requested")
				conn.HandleHttps(r)
			} else {
				log.Debug("HTTP Requested.")
				conn.HandleHttp(r)
			}
		}()
	}
}
