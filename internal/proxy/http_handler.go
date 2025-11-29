package proxy

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/applog"
	"github.com/xvzc/SpoofDPI/internal/proto"
)

type HTTPHandler struct {
	logger zerolog.Logger
}

func NewHTTPHandler(
	logger zerolog.Logger,
) *HTTPHandler {
	return &HTTPHandler{
		logger: logger,
	}
}

func (h *HTTPHandler) HandleRequest(
	ctx context.Context,
	lConn net.Conn, // Use the net.Conn interface, not a concrete *net.TCPConn.
	req *proto.HTTPRequest, // Assumes HttpRequest is a custom type for request parsing.
	dstAddrs []net.IPAddr,
	dstPort int,
	timeout time.Duration,
) error {
	logger := applog.WithLocalScope(h.logger, ctx, "http")

	rConn, err := dialFirstSuccessful(ctx, dstAddrs, dstPort, timeout)
	if err != nil {
		return err
	}

	// Ensure the remote connection is also closed on exit.
	defer closeConns(rConn)

	logger.Debug().Msgf("new remote conn -> %s", rConn.RemoteAddr())

	// Assumes our custom HttpRequest type has a WriteProxy method
	// (like net/http.Request.WriteProxy) that correctly formats the
	// request for the origin server (e.g., "GET /path" instead of "GET http://...").
	if err := req.WriteProxy(rConn); err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	errCh := make(chan error, 2)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go tunnelConns(ctx, logger, errCh, rConn, lConn)
	go tunnelConns(ctx, logger, errCh, lConn, rConn)

	for range 2 {
		e := <-errCh
		if e == nil {
			continue
		}

		return fmt.Errorf(
			"unsuccessful tunnel %s -> %s: %w",
			lConn.RemoteAddr(),
			rConn.RemoteAddr(),
			e,
		)
	}

	return nil
}
