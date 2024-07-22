package proxy

import (
	log "github.com/sirupsen/logrus"
	"github.com/xvzc/SpoofDPI/net"
	"github.com/xvzc/SpoofDPI/packet"
)

func (pxy *Proxy) HandleHttp(lConn *net.Conn, pkt *packet.HttpPacket, ip string) {
	pkt.Tidy()

	// Create connection to server
	var port = "80"
	if pkt.Port() != "" {
		port = pkt.Port()
	}

	rConn, err := net.DialTCP("tcp", ip, port)
	if err != nil {
    lConn.Close()
		log.Debug("[HTTP] ", err)
		return
	}

	defer func() {
		lConn.Close()
		log.Debug("[HTTP] Closing client Connection.. ", lConn.RemoteAddr())

		rConn.Close()
		log.Debug("[HTTP] Closing server Connection.. ", pkt.Domain(), " ", rConn.LocalAddr())
	}()

	log.Debug("[HTTP] New connection to the server ", pkt.Domain(), " ", rConn.LocalAddr())

	go rConn.Serve(lConn, "[HTTP]", lConn.RemoteAddr().String(), pkt.Domain(), pxy.timeout)

	_, err = rConn.Write(pkt.Raw())
	if err != nil {
		log.Debug("[HTTP] Error sending request to ", pkt.Domain(), err)
		return
	}

	log.Debug("[HTTP] Sent a request to ", pkt.Domain())

	lConn.Serve(rConn, "[HTTP]", lConn.RemoteAddr().String(), pkt.Domain(), pxy.timeout)
}
