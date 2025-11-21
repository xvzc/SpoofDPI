package proxy

import (
	"context"
	"net"
	"time"

	"github.com/rs/zerolog"
)

type HTTPHandler struct {
	logger zerolog.Logger
}

func NewHttpHandler(
	logger zerolog.Logger,
) *HTTPHandler {
	return &HTTPHandler{
		logger: logger,
	}
}

func (h *HTTPHandler) HandleRequest(
	ctx context.Context,
	lConn net.Conn, // Use the net.Conn interface, not a concrete *net.TCPConn.
	req *HttpRequest, // Assumes HttpRequest is a custom type for request parsing.
	domain string,
	dstAddrs []net.IPAddr,
	dstPort int,
	timeout time.Duration,
) {
	logger := h.logger.With().Ctx(ctx).Logger()

	// The client connection is always closed when Serve returns.
	defer closeConns(lConn) // Assumes closeConns is a nil-safe helper.

	rConn, err := dialFirstSuccessful(ctx, dstAddrs, dstPort, timeout)
	if err != nil {
		logger.Debug().Msgf("all dial attempts to %s failed: %s", domain, err)

		return
	}

	// Ensure the remote connection is also closed on exit.
	defer closeConns(rConn)

	logger.Debug().Msgf("new conn; http; %s -> %s(%s);",
		rConn.LocalAddr(), domain, rConn.RemoteAddr(),
	)

	// Assumes our custom HttpRequest type has a WriteProxy method
	// (like net/http.Request.WriteProxy) that correctly formats the
	// request for the origin server (e.g., "GET /path" instead of "GET http://...").
	if err := req.WriteProxy(rConn); err != nil {
		logger.Debug().Msgf("error sending request to %s: %v", domain, err)
		return
	}

	// Start the tunnel using the refactored helper function.
	go tunnel(ctx, logger, nil, rConn, lConn, domain, true)
	tunnel(ctx, logger, nil, lConn, rConn, domain, false)
}
