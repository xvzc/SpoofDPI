package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/logging"
	"github.com/xvzc/SpoofDPI/internal/netutil"
	"github.com/xvzc/SpoofDPI/internal/proto"
	"github.com/xvzc/SpoofDPI/internal/proxy/tlsutil"
)

type HTTPSHandler struct {
	logger zerolog.Logger
	bridge *tlsutil.TLSBridge
}

func NewHTTPSHandler(
	logger zerolog.Logger,
	bridge *tlsutil.TLSBridge,
) *HTTPSHandler {
	return &HTTPSHandler{
		logger: logger,
		bridge: bridge,
	}
}

func (h *HTTPSHandler) HandleRequest(
	ctx context.Context,
	lConn net.Conn,
	dst *netutil.Destination,
	rule *config.Rule,
) error {
	logger := logging.WithLocalScope(ctx, h.logger, "handshake")

	// 1. Send 200 Connection Established
	if err := proto.HTTPConnectionEstablishedResponse().Write(lConn); err != nil {
		if !netutil.IsConnectionResetByPeer(err) && !errors.Is(err, io.EOF) {
			logger.Trace().Err(err).Msgf("proxy handshake error")
			return fmt.Errorf("failed to handle proxy handshake: %w", err)
		}
		return nil
	}
	logger.Trace().Msgf("sent 200 connection established -> %s", lConn.RemoteAddr())

	// 2. Delegate to Bridge
	return h.bridge.Tunnel(ctx, lConn, dst, rule)
}

