package proxy

import (
	"context"
	"github.com/xvzc/SpoofDPI/util"
	"net"
	"strconv"

	"github.com/xvzc/SpoofDPI/util/log"

	"github.com/xvzc/SpoofDPI/packet"
)

const protoHTTP = "HTTP"

func (pxy *Proxy) handleHttp(ctx context.Context, lConn *net.TCPConn, pkt *packet.HttpRequest, ip string) {
	ctx = util.GetCtxWithScope(ctx, protoHTTP)
	logger := log.GetCtxLogger(ctx)

	pkt.Tidy()

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
