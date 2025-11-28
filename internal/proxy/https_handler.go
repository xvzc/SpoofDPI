package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/appctx"
	"github.com/xvzc/SpoofDPI/internal/applog"
	"github.com/xvzc/SpoofDPI/internal/desync"
	"github.com/xvzc/SpoofDPI/internal/proto"
)

var _ Handler = (*HTTPSHandler)(nil)

type HTTPSHandler struct {
	logger     zerolog.Logger
	tlsDefault desync.TLSDesyncer
	tlsBypass  desync.TLSDesyncer
}

func NewHTTPSHandler(
	logger zerolog.Logger,
	tlsDefault desync.TLSDesyncer,
	tlsBypass desync.TLSDesyncer,
) *HTTPSHandler {
	return &HTTPSHandler{
		logger:     logger,
		tlsDefault: tlsDefault,
		tlsBypass:  tlsBypass,
	}
}

func (h *HTTPSHandler) HandleRequest(
	ctx context.Context,
	lConn net.Conn,
	req *proto.HTTPRequest,
	dstAddrs []net.IPAddr,
	dstPort int,
	timeout time.Duration,
) error {
	logger := applog.WithLocalScope(h.logger, ctx, "https")

	rConn, err := dialFirstSuccessful(ctx, dstAddrs, dstPort, timeout)
	if err != nil {
		return err
	}
	defer closeConns(rConn)

	logger.Debug().
		Msgf("new remote conn -> %s", rConn.RemoteAddr())

	tlsMsg, err := h.handleProxyHandshake(ctx, lConn, req)
	if err != nil {
		logger.Trace().Err(err).Msgf("proxy handshake error")
		if !isConnectionResetByPeer(err) && !errors.Is(err, io.EOF) {
			return fmt.Errorf("failed to handle proxy handshake: %w", err)
		}

		return nil
	}

	if !tlsMsg.IsClientHello() {
		logger.Trace().Int("len", len(tlsMsg.Raw)).Msg("not a client hello. aborting")
		return nil
	}

	n, err := h.sendClientHello(ctx, rConn, tlsMsg)
	if err != nil {
		return fmt.Errorf("failed to send client hello: %w", err)
	}

	logger.Debug().
		Int("len", n).
		Msgf("sent client hello -> %s", rConn.RemoteAddr())

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

		if isConnectionResetByPeer(e) {
			return errBlocked
		} else {
			return fmt.Errorf(
				"unsuccessful tunnel %s -> %s: %w",
				lConn.RemoteAddr(),
				rConn.RemoteAddr(),
				e,
			)
		}
	}

	return nil
}

// handleProxyHandshake sends "200 Connection Established" and reads the subsequent Client Hello.
func (h *HTTPSHandler) handleProxyHandshake(
	ctx context.Context,
	lConn net.Conn,
	req *proto.HTTPRequest,
) (*proto.TLSMessage, error) {
	logger := applog.WithLocalScope(h.logger, ctx, "handshake")

	if _, err := lConn.Write(req.ResConnectionEstablished()); err != nil {
		return nil, err
	}
	logger.Trace().Msgf("sent 200 connection established -> %s", lConn.RemoteAddr())

	tlsMsg, err := proto.ReadTLSMessage(lConn)
	if err != nil {
		return nil, err
	}

	logger.Debug().
		Int("len", len(tlsMsg.Raw)).
		Msgf("client hello received <- %s", lConn.RemoteAddr())

	return tlsMsg, nil
}

// sendClientHello decides whether to spoof and sends the Client Hello accordingly.
func (h *HTTPSHandler) sendClientHello(
	ctx context.Context,
	conn net.Conn,
	msg *proto.TLSMessage,
) (int, error) {
	logger := applog.WithLocalScope(h.logger, ctx, "client_hello")

	var strategy desync.TLSDesyncer

	shouldExploit, ok := appctx.ShouldExploitFrom(ctx)
	if ok {
		if shouldExploit {
			strategy = h.tlsBypass
		} else {
			strategy = h.tlsDefault
		}
	} else {
		logger.Error().
			Str("key", "shouldExploit").
			Msgf("failed to retrieve value from ctx. default to `plain`")

		strategy = h.tlsDefault
	}

	logger.Debug().Msgf("using '%v' strategy", strategy)

	return strategy.Send(ctx, logger, conn, msg)
}
