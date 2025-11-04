package proxy

import (
	"context"
	"io"
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

func (h *HTTPHandler) Serve(
	ctx context.Context,
	lConn net.Conn, // Use the net.Conn interface, not the concrete *net.TCPConn
	req *HttpRequest, // Assuming HttpRequest is a custom type that handles request parsing
	domain string,
	dstAddrs []net.IPAddr,
	dstPort int,
	timeout time.Duration,
) {
	logger := h.logger.With().Ctx(ctx).Logger()

	// The client connection is *always* closed when Serve returns.
	defer closeConns(lConn) // Assumes closeConns is a nil-safe helper

	rConn, err := dialFirstSuccessful(ctx, dstAddrs, dstPort, timeout)
	if err != nil {
		logger.Debug().
			Msgf("dial to %s failed: %s", domain, err)
	}

	// Ensure the remote connection is also closed on exit.
	defer closeConns(rConn)
	logger.Debug().
		Msgf("new connection established %s -> %s", rConn.LocalAddr(), domain)

	// We assume our custom HttpRequest type has a WriteProxy method
	// (like net/http.Request.WriteProxy) that correctly formats the
	// request for the origin server (e.g., "GET /path" instead of "GET http://...").
	if err := req.WriteProxy(rConn); err != nil {
		logger.Debug().Msgf("error sending request to %s: %v", domain, err)
		return
	}

	// All custom concurrency logic is replaced by io.Copy.
	// HTTP is a request/response flow. After the request is sent,
	// We only need to copy the response back from the server to the client.
	// This one line replaces the complex, buggy copyData and sync logic.
	if _, err := io.Copy(lConn, rConn); err != nil {
		// io.EOF is a clean shutdown (the server finished sending data),
		// so it's not an error we need to log.
		if err != io.EOF {
			logger.Debug().Msgf("server->client copy error: %v", err)
		}
	}

	// When Serve returns, both lConn and rConn will be closed by their defers.
}
