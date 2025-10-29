package http

import (
	"context"
	"net"
	"strconv"

	"github.com/xvzc/SpoofDPI/config"
	"github.com/xvzc/SpoofDPI/packet"
	"github.com/xvzc/SpoofDPI/proxy/handler"
	"github.com/xvzc/SpoofDPI/util"
	"github.com/xvzc/SpoofDPI/util/log"
	"sync"
)

var lock = &sync.Mutex{}

type HttpHandler struct {
	bufferSize int
	port       int
}

var instance *HttpHandler

func GetInstance() *HttpHandler {
	if instance == nil {
		lock.Lock()
		defer lock.Unlock()
		if instance == nil {
			instance = &HttpHandler{
				bufferSize: 1024,
				port:       80,
			}
		}
	}

	return instance
}

func (h *HttpHandler) Protocol() string {
	return "HTTP"
}

func (h *HttpHandler) Serve(ctx context.Context, lConn *net.TCPConn, initPkt *packet.HttpRequest, ip string) {
	ctx = util.GetCtxWithScope(ctx, h.Protocol())
	logger := log.GetCtxLogger(ctx)

	// Create a connection to the requested server
	var port int = 80
	var err error
	if initPkt.Port() != "" {
		port, err = strconv.Atoi(initPkt.Port())
		if err != nil {
			logger.Debug().Msgf("error while parsing port for %s aborting..", initPkt.Domain())
		}
	}

	rConn, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: net.ParseIP(ip), Port: port})
	if err != nil {
		lConn.Close()
		logger.Debug().Msgf("%s", err)
		return
	}

	logger.Debug().Msgf("new connection to the server %s -> %s", rConn.LocalAddr(), initPkt.Domain())

	go h.deliverResponse(ctx, rConn, lConn, initPkt.Domain(), lConn.RemoteAddr().String())
	go h.deliverRequest(ctx, lConn, rConn, lConn.RemoteAddr().String(), initPkt.Domain())

	_, err = rConn.Write(initPkt.Raw())
	if err != nil {
		logger.Debug().Msgf("error sending request to %s: %s", initPkt.Domain(), err)
		return
	}
}

func (h *HttpHandler) deliverRequest(ctx context.Context, from *net.TCPConn, to *net.TCPConn, fd string, td string) {
	c := config.Get()
	ctx = util.GetCtxWithScope(ctx, h.Protocol())
	logger := log.GetCtxLogger(ctx)

	defer func() {
		from.Close()
		to.Close()

		logger.Debug().Msgf("closing proxy connection: %s -> %s", fd, td)
	}()

	for {
		err := handler.SetConnectionTimeout(from, c.Timeout())
		if err != nil {
			logger.Debug().Msgf("error while setting connection deadline for %s: %s", fd, err)
		}

		pkt, err := packet.ReadHttpRequest(from)
		if err != nil {
			logger.Debug().Msgf("error reading from %s: %s", fd, err)
			return
		}

		pkt.Tidy()

		if _, err := to.Write(pkt.Raw()); err != nil {
			logger.Debug().Msgf("error writing to %s", td)
			return
		}
	}
}

func (h *HttpHandler) deliverResponse(ctx context.Context, from *net.TCPConn, to *net.TCPConn, fd string, td string) {
	c := config.Get()
	ctx = util.GetCtxWithScope(ctx, h.Protocol())
	logger := log.GetCtxLogger(ctx)

	defer func() {
		from.Close()
		to.Close()

		logger.Debug().Msgf("closing proxy connection: %s -> %s", fd, td)
	}()

	buf := make([]byte, h.bufferSize)
	for {
		err := handler.SetConnectionTimeout(from, c.Timeout())
		if err != nil {
			logger.Debug().Msgf("error while setting connection deadline for %s: %s", fd, err)
		}

		bytesRead, err := handler.ReadBytes(ctx, from, buf)
		if err != nil {
			logger.Debug().Msgf("error reading from %s: %s", fd, err)
			return
		}

		if _, err := to.Write(bytesRead); err != nil {
			logger.Debug().Msgf("error writing to %s", td)
			return
		}
	}
}
