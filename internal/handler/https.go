package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/desync"
	"github.com/xvzc/SpoofDPI/internal/logging"
	"github.com/xvzc/SpoofDPI/internal/netutil"
	"github.com/xvzc/SpoofDPI/internal/packet"
	"github.com/xvzc/SpoofDPI/internal/proto"
	"github.com/xvzc/SpoofDPI/internal/ptr"
)

var _ RequestHandler = (*HTTPSHandler)(nil)

type HTTPSHandler struct {
	logger     zerolog.Logger
	desyncer   *desync.TLSDesyncer
	hopTracker *packet.HopTracker
	httpsOpts  *config.HTTPSOptions
}

func NewHTTPSHandler(
	logger zerolog.Logger,
	desyncer *desync.TLSDesyncer,
	hopTracker *packet.HopTracker,
	httpsOpts *config.HTTPSOptions,
) *HTTPSHandler {
	return &HTTPSHandler{
		logger:     logger,
		desyncer:   desyncer,
		hopTracker: hopTracker,
		httpsOpts:  httpsOpts,
	}
}

//	func (h *HTTPSHandler) DefaultRule() *policy.Rule {
//		return h.defaultAttrs.Clone()
//	}
func (h *HTTPSHandler) HandleRequest(
	ctx context.Context,
	lConn net.Conn,
	req *proto.HTTPRequest,
	dst *Destination,
	rule *config.Rule,
) error {
	httpsOpts := h.httpsOpts
	if rule != nil {
		httpsOpts = httpsOpts.Merge(rule.HTTPS)
	}

	if h.hopTracker != nil && ptr.FromPtr(httpsOpts.FakeCount) > 0 {
		h.hopTracker.RegisterUntracked(dst.Addrs, dst.Port)
	}

	logger := logging.WithLocalScope(ctx, h.logger, "https")

	rConn, err := netutil.DialFirstSuccessful(ctx, dst.Addrs, dst.Port, dst.Timeout)
	if err != nil {
		return err
	}
	defer netutil.CloseConns(rConn)

	logger.Debug().Msgf("new remote conn -> %s", rConn.RemoteAddr())

	tlsMsg, err := h.handleProxyHandshake(ctx, lConn, req)
	if err != nil {
		if !netutil.IsConnectionResetByPeer(err) && !errors.Is(err, io.EOF) {
			logger.Trace().Err(err).Msgf("proxy handshake error")
			return fmt.Errorf("failed to handle proxy handshake: %w", err)
		}

		return nil
	}

	if !tlsMsg.IsClientHello() {
		logger.Trace().Int("len", len(tlsMsg.Raw)).Msg("not a client hello. aborting")
		return nil
	}

	n, err := h.sendClientHello(ctx, rConn, tlsMsg, httpsOpts)
	if err != nil {
		return fmt.Errorf("failed to send client hello: %w", err)
	}

	logger.Debug().
		Int("len", n).
		Msgf("sent client hello -> %s", rConn.RemoteAddr())

	errCh := make(chan error, 2)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go netutil.TunnelConns(ctx, logger, errCh, rConn, lConn)
	go netutil.TunnelConns(ctx, logger, errCh, lConn, rConn)

	for range 2 {
		e := <-errCh
		if e == nil {
			continue
		}

		if netutil.IsConnectionResetByPeer(e) {
			return netutil.ErrBlocked
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

// handleProxyHandshake sends "200 Connection Established"
// and reads the subsequent Client Hello.
func (h *HTTPSHandler) handleProxyHandshake(
	ctx context.Context,
	lConn net.Conn,
	req *proto.HTTPRequest,
) (*proto.TLSMessage, error) {
	logger := logging.WithLocalScope(ctx, h.logger, "handshake")

	if _, err := lConn.Write(req.ConnEstablishedResponse()); err != nil {
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
	httpsOpts *config.HTTPSOptions,
) (int, error) {
	logger := logging.WithLocalScope(ctx, h.logger, "client_hello")
	return h.desyncer.Send(ctx, logger, msg, conn, httpsOpts)
}
