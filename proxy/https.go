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
			log.Debug("[HTTPS] error while parsing port for ", initPkt.Domain(), " aborting..")
		}
	}

	rConn, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: net.ParseIP(ip), Port: port})
	if err != nil {
		lConn.Close()
		log.Debug("[HTTPS] ", err)
		return
	}

	log.Debug("[HTTPS] new connection to the server ", rConn.LocalAddr(), " -> ", initPkt.Domain())

	_, err = lConn.Write([]byte(initPkt.Version() + " 200 Connection Established\r\n\r\n"))
	if err != nil {
		log.Debug("[HTTPS] error sending 200 connection established to the client: ", err)
		return
	}

	log.Debug("[HTTPS] sent connection estabalished to ", lConn.RemoteAddr())

	// Read client hello
	m, err := packet.ReadTLSMessage(lConn)
	if err != nil || !m.IsClientHello() {
		log.Debug("[HTTPS] error reading client hello from ", lConn.RemoteAddr().String(), " ", err)
		return
	}
	clientHello := m.Raw

	log.Debug("[HTTPS] client sent hello ", len(clientHello), "bytes")

	// Generate a go routine that reads from the server
	go Serve(rConn, lConn, "[HTTPS]", initPkt.Domain(), lConn.RemoteAddr().String(), pxy.timeout)

	if exploit {
		log.Debug("[HTTPS] writing chunked client hello to ", initPkt.Domain())
		chunks := splitInChunks(clientHello, pxy.windowSize)
		if _, err := writeChunks(rConn, chunks); err != nil {
			log.Debug("[HTTPS] error writing chunked client hello to ", initPkt.Domain(), err)
			return
		}
	} else {
		log.Debug("[HTTPS] writing plain client hello to ", initPkt.Domain())
		if _, err := rConn.Write(clientHello); err != nil {
			log.Debug("[HTTPS] error writing plain client hello to ", initPkt.Domain(), err)
			return
		}
	}

	go Serve(lConn, rConn, "[HTTPS]", lConn.RemoteAddr().String(), initPkt.Domain(), pxy.timeout)
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

	log.Debug("[HTTPS] using legacy fragmentation")

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
