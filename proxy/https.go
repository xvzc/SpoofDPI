package proxy

import (
	"context"
	"fmt"
	"github.com/xvzc/SpoofDPI/packet"
	"github.com/xvzc/SpoofDPI/util"
	"github.com/xvzc/SpoofDPI/util/log"
	"net"
	"strconv"
	"strings"
	"time"
)

const protoHTTPS = "HTTPS"

func (pxy *Proxy) handleHttps(ctx context.Context, lConn *net.TCPConn, exploit bool, initPkt *packet.HttpRequest, ip string) {
	ctx = util.GetCtxWithScope(ctx, protoHTTPS)
	logger := log.GetCtxLogger(ctx)

	//Is proxy auth enable
	if pxy.proxyAuth != "" {
		addr := lConn.LocalAddr().String()

		var isAllow = false
		var cachedAuthData *Auth
		size := AuthorizedClientsCache.Size()

		for i := 0; i < size; i++ {
			cachedAuthData = AuthorizedClientsCache.At(i)
			if cachedAuthData.Addr == addr {
				isAllow = (time.Now().Unix() - cachedAuthData.AuthTime) < 60*30 //Cache available 30 min
				break
			}
		}

		var raw = string(initPkt.Raw())
		auth := extractProxyAuthorization(raw)

		var isAvailableAuthData = strings.TrimSpace(pxy.proxyAuth) == strings.TrimSpace(auth)

		if !isAllow && !isAvailableAuthData {
			response := []byte(fmt.Sprintf(initPkt.Version() + " 407 Proxy Authentication Required\nProxy-Authenticate: Basic realm=\"Access to internal site\"\r\n\r\n"))
			lConn.Write(response)
			lConn.Close()
			logger.Debug().Msgf("Unauthorized client: " + addr)
			return
		}

		go func() {
			if isAvailableAuthData && cachedAuthData != nil {
				cachedAuthData.AuthTime = time.Now().Unix()
			} else if isAvailableAuthData {
				var nA = Auth{}
				nA.Addr = addr
				nA.AuthTime = time.Now().Unix()
				AuthorizedClientsCache.Add(nA)
			}
		}()
	}

	// Create a connection to the requested server
	var port int = 443
	var err error
	if initPkt.Port() != "" {
		port, err = strconv.Atoi(initPkt.Port())
		if err != nil {
			logger.Debug().Msgf("error parsing port for %s aborting..", initPkt.Domain())
		}
	}

	rConn, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: net.ParseIP(ip), Port: port})
	if err != nil {
		lConn.Close()
		logger.Debug().Msgf("%s", err)
		return
	}

	logger.Debug().Msgf("new connection to the server %s -> %s", rConn.LocalAddr(), initPkt.Domain())

	_, err = lConn.Write([]byte(initPkt.Version() + " 200 Connection Established\r\n\r\n"))
	if err != nil {
		logger.Debug().Msgf("error sending 200 connection established to the client: %s", err)
		return
	}

	logger.Debug().Msgf("sent connection estabalished to %s", lConn.RemoteAddr())

	// Read client hello
	m, err := packet.ReadTLSMessage(lConn)
	if err != nil || !m.IsClientHello() {
		logger.Debug().Msgf("error reading client hello from %s: %s", lConn.RemoteAddr().String(), err)
		return
	}
	clientHello := m.Raw

	logger.Debug().Msgf("client sent hello %d bytes", len(clientHello))

	// Generate a go routine that reads from the server
	go Serve(ctx, rConn, lConn, protoHTTPS, initPkt.Domain(), lConn.RemoteAddr().String(), pxy.timeout)

	if exploit {
		logger.Debug().Msgf("writing chunked client hello to %s", initPkt.Domain())
		chunks := splitInChunks(ctx, clientHello, pxy.windowSize)
		if _, err := writeChunks(rConn, chunks); err != nil {
			logger.Debug().Msgf("error writing chunked client hello to %s: %s", initPkt.Domain(), err)
			return
		}
	} else {
		logger.Debug().Msgf("writing plain client hello to %s", initPkt.Domain())
		if _, err := rConn.Write(clientHello); err != nil {
			logger.Debug().Msgf("error writing plain client hello to %s: %s", initPkt.Domain(), err)
			return
		}
	}

	go Serve(ctx, lConn, rConn, protoHTTPS, lConn.RemoteAddr().String(), initPkt.Domain(), pxy.timeout)
}

func splitInChunks(ctx context.Context, bytes []byte, size int) [][]byte {
	logger := log.GetCtxLogger(ctx)

	var chunks [][]byte
	var raw []byte = bytes

	logger.Debug().Msgf("window-size: %d", size)

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

	logger.Debug().Msg("using legacy fragmentation")

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
