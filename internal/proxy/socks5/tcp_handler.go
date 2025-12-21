package socks5

import (
	"context"
	"errors"
	"net"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/netutil"
	"github.com/xvzc/SpoofDPI/internal/proto"
	"github.com/xvzc/SpoofDPI/internal/proxy/http"
)

type TCPHandler struct {
	logger       zerolog.Logger
	httpsHandler *http.HTTPSHandler
	serverOpts   *config.ServerOptions
}

func NewTCPHandler(
	logger zerolog.Logger,
	httpsHandler *http.HTTPSHandler,
	serverOpts *config.ServerOptions,
) *TCPHandler {
	return &TCPHandler{
		logger:       logger,
		httpsHandler: httpsHandler,
		serverOpts:   serverOpts,
	}
}

func (h *TCPHandler) Handle(
	ctx context.Context,
	conn net.Conn,
	req *proto.SOCKS5Request,
	dst *netutil.Destination,
	rule *config.Rule,
) error {
	logger := h.logger.With().Ctx(ctx).Logger()

	// 1. Validate Destination (Avoid Recursive Loop)
	ok, err := netutil.ValidateDestination(dst.Addrs, dst.Port, h.serverOpts.ListenAddr)
	if err != nil {
		logger.Debug().Err(err).Msg("error determining if valid destination")
		if !ok {
			_ = proto.SOCKS5FailureResponse().Write(conn)
			return err
		}
	}

	// 2. Check if blocked
	if rule != nil && *rule.Block {
		logger.Debug().Msg("request is blocked by policy")
		_ = proto.SOCKS5FailureResponse().Write(conn)
		return netutil.ErrBlocked
	}

	// 3. Send Success Response Optimistically
	if err := proto.SOCKS5SuccessResponse().Bind(net.IPv4zero).Port(0).Write(conn); err != nil {
		logger.Error().Err(err).Msg("failed to write socks5 success reply")
		return err
	}

	// 4. Handover to HTTPSHandler
	handleErr := h.httpsHandler.HandleRequest(ctx, conn, dst, rule)
	if handleErr == nil {
		return nil
	}

	logger.Warn().Err(handleErr).Msg("error handling request")
	if !errors.Is(handleErr, netutil.ErrBlocked) {
		return handleErr
	}

	return nil
}