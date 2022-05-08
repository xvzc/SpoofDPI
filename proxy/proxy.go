package proxy

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/xvzc/SpoofDPI/net"
	"github.com/xvzc/SpoofDPI/packet"
)

type Proxy struct {
	port string
    addr string
}

func New(addr string, port string) *Proxy {
	return &Proxy{
        addr: addr,
		port: port,
	}
}

func (p *Proxy) TcpAddr() string {
    return p.addr + ":" + p.port
}

func (p *Proxy) Port() string {
	return p.port
}

func (p *Proxy) Start() {
	l, err := net.Listen("tcp", p.TcpAddr())
	if err != nil {
		log.Fatal("Error creating listener: ", err)
		os.Exit(1)
	}

	log.Println("Created a listener on :", p.Port())

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal("Error accepting connection: ", err)
			continue
		}
        // conn.SetDeadLine(time.Now().Add(3 * time.Second))
        // conn.SetKeepAlive(false)

		go func() {
			b, err := conn.ReadBytes()
			if err != nil {
				return
			}

            log.Debug("[PROXY] Request from ", conn.RemoteAddr(), "\n\n", string(b))

			pkt, err := packet.NewHttpPacket(b)
            if err != nil {
                log.Debug("Error while parsing request: ", string(b))
                return
            }

			if !pkt.IsValidMethod() {
				log.Debug("Unsupported method: ", pkt.Method())
				return
			}

			if pkt.IsConnectMethod() {
				log.Debug("[HTTPS] Start")
				conn.HandleHttps(pkt)
			} else {
				log.Debug("[HTTP] Start")
				conn.HandleHttp(pkt)
			}
		}()
	}
}
