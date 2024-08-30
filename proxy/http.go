package proxy

import (
	"context"
	"fmt"
	"github.com/xvzc/SpoofDPI/util"
	"github.com/xvzc/SpoofDPI/util/log"
	"net"
	"strconv"

	"strings"
	"time"

	"github.com/xvzc/SpoofDPI/packet"
)

const protoHTTP = "HTTP"

func (pxy *Proxy) handleHttp(ctx context.Context, lConn *net.TCPConn, pkt *packet.HttpRequest, ip string) {
	ctx = util.GetCtxWithScope(ctx, protoHTTP)
	logger := log.GetCtxLogger(ctx)

	pkt.Tidy()

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

		var raw = string(pkt.Raw())
		auth := extractProxyAuthorization(raw)

		var isAvailableAuthData = strings.TrimSpace(pxy.proxyAuth) == strings.TrimSpace(auth)

		if !isAllow && !isAvailableAuthData {
			response := []byte(fmt.Sprintf(pkt.Version() + " 407 Proxy Authentication Required\nProxy-Authenticate: Basic realm=\"Access to internal site\"\r\n\r\n"))
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
	var port int = 80
	var err error
	if pkt.Port() != "" {
		port, err = strconv.Atoi(pkt.Port())
		if err != nil {
			logger.Debug().Msgf("error while parsing port for %s aborting..", pkt.Domain())
		}
	}

	rConn, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: net.ParseIP(ip), Port: port})
	if err != nil {
		lConn.Close()
		logger.Debug().Msgf("%s", err)
		return
	}

	logger.Debug().Msgf("new connection to the server %s -> %s", rConn.LocalAddr(), pkt.Domain())

	go Serve(ctx, rConn, lConn, protoHTTP, pkt.Domain(), lConn.RemoteAddr().String(), pxy.timeout)

	_, err = rConn.Write(pkt.Raw())
	if err != nil {
		logger.Debug().Msgf("error sending request to %s: %s", pkt.Domain(), err)
		return
	}

	logger.Debug().Msgf("sent a request to %s", pkt.Domain())

	go Serve(ctx, lConn, rConn, protoHTTP, lConn.RemoteAddr().String(), pkt.Domain(), pxy.timeout)
}

func extractProxyAuthorization(text string) string {
	start := strings.Index(text, "Proxy-Authorization: ")
	if start == -1 {
		return ""
	}

	end := strings.IndexByte(text[start:], '\n')
	if end == -1 {
		end = len(text)
	} else {
		end += start
	}

	authLine := text[start:end]

	authValue := strings.TrimPrefix(authLine, "Proxy-Authorization: ")

	parts := strings.SplitN(authValue, " ", 2)
	if len(parts) == 2 && parts[0] == "Basic" {
		return parts[1]
	}

	return ""
}
