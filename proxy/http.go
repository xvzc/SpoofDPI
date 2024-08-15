package proxy

import (
	"net"
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/xvzc/SpoofDPI/packet"
)

func (pxy *Proxy) handleHttp(lConn *net.TCPConn, pkt *packet.HttpPacket, ip string) {
	pkt.Tidy()

	// Create a connection to the requested server
	var port int = 80
	var err error
	if pkt.Port() != "" {
		port, err = strconv.Atoi(pkt.Port())
		if err != nil {
			log.Debug("[HTTP] error while parsing port for ", pkt.Domain(), " aborting..")
		}
	}

	rConn, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: net.ParseIP(ip), Port: port})
	if err != nil {
		lConn.Close()
		log.Debug("[HTTP] ", err)
		return
	}

	log.Debug("[HTTP] new connection to the server ", rConn.LocalAddr(), " -> ", pkt.Domain())

	go Serve(rConn, lConn, "[HTTP]", pkt.Domain(), lConn.RemoteAddr().String(), pxy.timeout)

	_, err = rConn.Write(pkt.Raw())
	if err != nil {
		log.Debug("[HTTP] error sending request to ", pkt.Domain(), err)
		return
	}

	log.Debug("[HTTP] sent a request to ", pkt.Domain())

	go Serve(lConn, rConn, "[HTTP]", lConn.RemoteAddr().String(), pkt.Domain(), pxy.timeout)
}
