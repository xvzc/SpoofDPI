package proxy

import (
	"net"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/xvzc/SpoofDPI/packet"
)

func (pxy *Proxy) handleHttps(lConn *net.TCPConn, exploit bool, initPkt *packet.HttpPacket, ip string) {
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
	m, err := packet.ReadTLSMessage(lConn)
	if err != nil || !m.IsClientHello() {
		log.Debug("[HTTPS] Error reading client hello from the client", err)
		return
	}
	clientHello := m.Raw

	log.Debug("[HTTPS] Client sent hello ", len(clientHello), "bytes")

	// Generate a go routine that reads from the server
	go Serve(rConn, lConn, "[HTTPS]", rConn.RemoteAddr().String(), initPkt.Domain(), pxy.timeout, pxy.bufferSize)

	if exploit {
		log.Debug("[HTTPS] Writing chunked client hello to ", initPkt.Domain())
		chunks := splitInChunks(clientHello, pxy.windowSize)
		if _, err := writeChunks(rConn, chunks); err != nil {
			log.Debug("[HTTPS] Error writing chunked client hello to ", initPkt.Domain(), err)
			return
		}
	} else {
		log.Debug("[HTTPS] Writing plain client hello to ", initPkt.Domain())
		if _, err := rConn.Write(clientHello); err != nil {
			log.Debug("[HTTPS] Error writing plain client hello to ", initPkt.Domain(), err)
			return
		}
	}

	Serve(lConn, rConn, "[HTTPS]", lConn.RemoteAddr().String(), initPkt.Domain(), pxy.timeout, pxy.bufferSize)
}

func splitInChunks(bytes []byte, size int) [][]byte {
	var chunks [][]byte
	var raw []byte = bytes

	log.Debug("[HTTPS] window-size: ", size)

	if size > 0 {
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

	// When the given window-size <= 0

	if len(raw) < 1 {
		return [][]byte{raw}
	}

	log.Debug("[HTTPS] Using legacy fragmentation.")

	return [][]byte{raw[:1], raw[1:]}
}

func writeChunks(conn *net.TCPConn, c [][]byte) (n int, err error) {
	total := 0
	for i := 0; i < len(c); i++ {
		b, err := conn.Write(c[i])
		if err != nil {
			return 0, nil
		}

		total += b
	}

	return total, nil
}

