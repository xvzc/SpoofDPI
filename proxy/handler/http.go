package handler

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/xvzc/SpoofDPI/packet"
	"github.com/xvzc/SpoofDPI/util"
	"github.com/xvzc/SpoofDPI/util/log"
)

type HttpHandler struct {
	bufferSize int
	protocol   string
	port       int
	timeout    int
}

func NewHttpHandler(timeout int) *HttpHandler {
	return &HttpHandler{
		bufferSize: 1024,
		protocol:   "HTTP",
		port:       80,
		timeout:    timeout,
	}
}

func (h *HttpHandler) Serve(ctx context.Context, lConn *net.TCPConn, pkt *packet.HttpRequest, ip string) {
	ctx = util.GetCtxWithScope(ctx, h.protocol)
	logger := log.GetCtxLogger(ctx)

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

	go h.deliverResponse(ctx, rConn, lConn, pkt.Domain(), lConn.RemoteAddr().String())
	go h.deliverRequest(ctx, lConn, rConn, lConn.RemoteAddr().String(), pkt.Domain())

	_, err = rConn.Write(pkt.Raw())
	if err != nil {
		logger.Debug().Msgf("error sending request to %s: %s", pkt.Domain(), err)
		return
	}
}

func (h *HttpHandler) deliverRequest(
	ctx context.Context,
	lConn *net.TCPConn,
	rConn *net.TCPConn,
	fd string,
	td string,
) {
	ctx = util.GetCtxWithScope(ctx, h.protocol)
	logger := log.GetCtxLogger(ctx)

	defer func() {
		lConn.Close()
		rConn.Close()

		logger.Debug().Msgf("closing proxy connection: %s -> %s", fd, td)
	}()

	if h.timeout > 0 {
		lConn.SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(h.timeout)))
	}

	for {
		pkt, err := packet.ReadHttpRequest(lConn)
		if err != nil {
			logger.Debug().Msgf("error reading from %s: %s", fd, err)
			return
		}

		pkt.Tidy()

		if _, err := rConn.Write(pkt.Raw()); err != nil {
			logger.Debug().Msgf("error Writing to %s", td)
			return
		}
	}
}

func (h *HttpHandler) deliverResponse(
	ctx context.Context,
	from *net.TCPConn,
	to *net.TCPConn,
	fd string,
	td string,
) {
	ctx = util.GetCtxWithScope(ctx, h.protocol)
	logger := log.GetCtxLogger(ctx)

	defer func() {
		from.Close()
		to.Close()

		logger.Debug().Msgf("closing proxy connection: %s -> %s", fd, td)
	}()

	buf := make([]byte, h.bufferSize)
	for {
		if h.timeout > 0 {
			from.SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(h.timeout)))
		}

		bytesRead, err := ReadBytes(ctx, from, buf)
		if err != nil {
			logger.Debug().Msgf("error reading from %s: %s", fd, err)
			return
		}

		if _, err := to.Write(bytesRead); err != nil {
			logger.Debug().Msgf("error Writing to %s", td)
			return
		}
	}
}
