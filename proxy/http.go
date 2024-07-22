package proxy

import (
	"net"
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/xvzc/SpoofDPI/packet"
)

func (pxy *Proxy) HandleHttp(lConn *net.TCPConn, pkt *packet.HttpPacket, ip string) {
	pkt.Tidy()

	// Create a connection to the requested server
	var port int = 80
	var err error
	if pkt.Port() != "" {
		port, err = strconv.Atoi(pkt.Port())
		if err != nil {
			log.Debug("[HTTP] Error while parsing port for ", pkt.Domain(), " aborting..")
		}
	}

	rConn, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: net.ParseIP(ip), Port: port})
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

	go Serve(rConn, lConn, "[HTTP]", lConn.RemoteAddr().String(), pkt.Domain(), pxy.timeout)

	_, err = rConn.Write(pkt.Raw())
	if err != nil {
		log.Debug("[HTTP] Error sending request to ", pkt.Domain(), err)
		return
	}

	log.Debug("[HTTP] Sent a request to ", pkt.Domain())

	Serve(lConn, rConn, "[HTTP]", lConn.RemoteAddr().String(), pkt.Domain(), pxy.timeout)
}
