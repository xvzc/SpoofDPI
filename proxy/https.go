package proxy

import (
	"net"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/xvzc/SpoofDPI/packet"
)

func (pxy *Proxy) handleHttps(lConn *net.TCPConn, initPkt *packet.HttpPacket, ip string) {
	// Create a connection to the requested server
	var port int = 443
	var err error
	if initPkt.Port() != "" {
		port, err = strconv.Atoi(initPkt.Port())
		if err != nil {
			log.Debug("[HTTPS] Error while parsing port for ", initPkt.Domain(), " aborting..")
		}
	}

	rConn, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: net.ParseIP(ip), Port: port})
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
	clientHello, err := ReadBytes(lConn)
	if err != nil {
		log.Debug("[HTTPS] Error reading client hello from the client", err)
		return
	}

	log.Debug("[HTTPS] Client sent hello ", len(clientHello), "bytes")

	// Generate a go routine that reads from the server

	chPkt := packet.NewHttpsPacket(clientHello)

	// lConn.SetLinger(3)
	// rConn.SetLinger(3)

	go Serve(rConn, lConn, "[HTTPS]", rConn.RemoteAddr().String(), initPkt.Domain(), pxy.timeout)

	if pxy.patternExists() && !pxy.patternMatches([]byte(initPkt.Domain())) {
		log.Debug("[HTTPS] Writing plain client hello to ", initPkt.Domain())
		if _, err := rConn.Write(chPkt.Raw()); err != nil {
			log.Debug("[HTTPS] Error writing plain client hello to ", initPkt.Domain(), err)
			return
		}
	} else {
		log.Debug("[HTTPS] Writing chunked client hello to ", initPkt.Domain())
		chunks := pxy.splitInChunks(chPkt.Raw(), pxy.windowSize)
		if _, err := WriteChunks(rConn, chunks); err != nil {
			log.Debug("[HTTPS] Error writing chunked client hello to ", initPkt.Domain(), err)
			return
		}
	}

	Serve(lConn, rConn, "[HTTPS]", lConn.RemoteAddr().String(), initPkt.Domain(), pxy.timeout)
}

func (pxy *Proxy) splitInChunks(bytes []byte, size int) [][]byte {
	// If the packet matches the pattern or the URLs, we don't split it
	if pxy.patternExists() && !pxy.patternMatches(bytes) {
		return [][]byte{bytes}
	}

	var chunks [][]byte
	var raw []byte = bytes

	if pxy.windowSize > 0 {
    log.Debug("[HTTPS] Chunking client hello.")

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

	if len(raw) < 1 {
		return [][]byte{raw}
	}

  log.Debug("[HTTPS] Using legacy fragmentation.")

	return [][]byte{raw[:1], raw[1:]}
}

func (pxy *Proxy) patternExists() bool {
	return pxy.allowedPattern != nil || pxy.allowedUrls != nil
}

func (pxy *Proxy) patternMatches(bytes []byte) bool {
	return (pxy.allowedPattern != nil && pxy.allowedPattern.Match(bytes)) ||
		(pxy.allowedUrls != nil && pxy.allowedUrls.Match(bytes))
}
