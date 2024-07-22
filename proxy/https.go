package proxy

import (
	log "github.com/sirupsen/logrus"
	"github.com/xvzc/SpoofDPI/net"
	"github.com/xvzc/SpoofDPI/packet"
)

func (pxy *Proxy) HandleHttps(lConn *net.Conn, initPkt *packet.HttpPacket, ip string) {
	// Create a connection to the requested server
	var port = "443"
	if initPkt.Port() != "" {
		port = initPkt.Port()
	}

	rConn, err := net.DialTCP("tcp4", ip, port)
	if err != nil {
    lConn.Close()
		log.Debug("[HTTPS] ", err)
		return
	}

	defer func() {
		lConn.Close()
		log.Debug("[HTTPS] Closing client Connection.. ", lConn.RemoteAddr())

		rConn.Close()
		log.Debug("[HTTPS] Closing server Connection.. ", initPkt.Domain(), " ", rConn.LocalAddr())
	}()

	log.Debug("[HTTPS] New connection to the server ", initPkt.Domain(), " ", rConn.LocalAddr())

	_, err = lConn.Write([]byte(initPkt.Version() + " 200 Connection Established\r\n\r\n"))
	if err != nil {
		log.Debug("[HTTPS] Error sending 200 Connection Established to the client", err)
		return
	}

	log.Debug("[HTTPS] Sent 200 Connection Estabalished to ", lConn.RemoteAddr())

	// Read client hello
	clientHello, err := lConn.ReadBytes()
	if err != nil {
		log.Debug("[HTTPS] Error reading client hello from the client", err)
		return
	}

	log.Debug("[HTTPS] Client sent hello ", len(clientHello), "bytes")

	// Generate a go routine that reads from the server

	chPkt := packet.NewHttpsPacket(clientHello)

	go rConn.Serve(lConn, "[HTTPS]", rConn.RemoteAddr().String(), initPkt.Domain(), pxy.timeout)

	if pxy.patternExists() && !pxy.patternMatches([]byte(initPkt.Domain())) {
    if _, err := rConn.Write(chPkt.Raw()); err != nil {
      log.Debug("[HTTPS] Error writing client hello to ", initPkt.Domain(), err)
      return
    }
	} else {
    chunks := pxy.splitInChunks(chPkt.Raw(), pxy.windowSize)
    if _, err := rConn.WriteChunks(chunks); err != nil {
      log.Debug("[HTTPS] Error writing client hello to ", initPkt.Domain(), err)
      return
    }
  }

	lConn.Serve(rConn, "[HTTPS]", lConn.RemoteAddr().String(), initPkt.Domain(), pxy.timeout)
}

func (pxy *Proxy) splitInChunks(bytes []byte, size int) [][]byte {
	// If the packet matches the pattern or the URLs, we don't split it
	if pxy.patternExists() && !pxy.patternMatches(bytes) {
		return [][]byte{bytes}
	}

	var chunks [][]byte
	var raw []byte = bytes

	for {
		if len(raw) == 0 {
			break
		}

		// necessary check to avoid slicing beyond
		// slice capacity
		if len(raw) < size {
			size = len(raw)
		}

		chunks = append(chunks, raw[0:size])
		raw = raw[size:]
	}

	return chunks
}

func (pxy *Proxy) patternExists() bool {
	return pxy.allowedPattern != nil || pxy.allowedUrls != nil
}

func (p *Proxy) patternMatches(bytes []byte) bool {
	return (p.allowedPattern != nil && p.allowedPattern.Match(bytes)) ||
		(p.allowedUrls != nil && p.allowedUrls.Match(bytes))
}
