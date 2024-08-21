package proxy

import (
	"context"
	"net"
	"strconv"

	"github.com/xvzc/SpoofDPI/packet"
	"github.com/xvzc/SpoofDPI/util/log"
)

const protoHTTPS = "HTTPS"

func (pxy *Proxy) handleHttps(ctx context.Context, lConn *net.TCPConn, exploit bool, initPkt *packet.HttpRequest, ip string) {
	// Create a connection to the requested server
	var port int = 443
	var err error
	if initPkt.Port() != "" {
		port, err = strconv.Atoi(initPkt.Port())
		if err != nil {
			log.Logger.Debug().
				Str(log.ScopeFieldName, protoHTTPS).
				Msgf("error parsing port for %s aborting..", initPkt.Domain())
		}
	}

	rConn, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: net.ParseIP(ip), Port: port})
	if err != nil {
		lConn.Close()
		log.Logger.Debug().
			Str(log.ScopeFieldName, protoHTTPS).
			Msgf("%s", err)
		return
	}

	log.Logger.Debug().
		Str(log.ScopeFieldName, protoHTTPS).
		Msgf("new connection to the server %s -> %s", rConn.LocalAddr(), initPkt.Domain())

	_, err = lConn.Write([]byte(initPkt.Version() + " 200 Connection Established\r\n\r\n"))
	if err != nil {
		log.Logger.Debug().
			Str(log.ScopeFieldName, protoHTTPS).
			Msgf("error sending 200 connection established to the client: %s", err)
		return
	}

	log.Logger.Debug().
		Str(log.ScopeFieldName, protoHTTPS).
		Msgf("sent connection estabalished to %s", lConn.RemoteAddr())

	// Read client hello
	m, err := packet.ReadTLSMessage(lConn)
	if err != nil || !m.IsClientHello() {
		log.Logger.Debug().
			Str(log.ScopeFieldName, protoHTTPS).
			Msgf("error reading client hello from %s: %s", lConn.RemoteAddr().String(), err)
		return
	}
	clientHello := m.Raw

	log.Logger.Debug().
		Str(log.ScopeFieldName, protoHTTPS).
		Msgf("client sent hello %d bytes", len(clientHello))

	// Generate a go routine that reads from the server
	go Serve(ctx, rConn, lConn, protoHTTPS, initPkt.Domain(), lConn.RemoteAddr().String(), pxy.timeout)

	if exploit {
		log.Logger.Debug().
			Str(log.ScopeFieldName, protoHTTPS).
			Msgf("writing chunked client hello to %s", initPkt.Domain())
		chunks := splitInChunks(ctx, clientHello, pxy.windowSize)
		if _, err := writeChunks(rConn, chunks); err != nil {
			log.Logger.Debug().
				Str(log.ScopeFieldName, protoHTTPS).
				Msgf("error writing chunked client hello to %s: %s", initPkt.Domain(), err)
			return
		}
	} else {
		log.Logger.Debug().
			Str(log.ScopeFieldName, protoHTTPS).
			Msgf("writing plain client hello to %s", initPkt.Domain())
		if _, err := rConn.Write(clientHello); err != nil {
			log.Logger.Debug().
				Str(log.ScopeFieldName, protoHTTPS).
				Msgf("error writing plain client hello to %s: %s", initPkt.Domain(), err)
			return
		}
	}

	go Serve(ctx, lConn, rConn, protoHTTPS, lConn.RemoteAddr().String(), initPkt.Domain(), pxy.timeout)
}

func splitInChunks(ctx context.Context, bytes []byte, size int) [][]byte {
	var chunks [][]byte
	var raw []byte = bytes

	log.Logger.Debug().
		Str(log.ScopeFieldName, protoHTTPS).
		Msgf("window-size: %d", size)

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

	log.Logger.Debug().
		Str(log.ScopeFieldName, protoHTTPS).
		Msg("using legacy fragmentation")

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
