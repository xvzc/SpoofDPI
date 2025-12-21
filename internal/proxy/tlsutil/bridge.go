package tlsutil

import (
	"context"
	"fmt"
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

type TLSBridge struct {
	logger    zerolog.Logger
	desyncer  *desync.TLSDesyncer
	sniffer   packet.Sniffer
	httpsOpts *config.HTTPSOptions
}

func NewTLSBridge(
	logger zerolog.Logger,
	desyncer *desync.TLSDesyncer,
	sniffer packet.Sniffer,
	httpsOpts *config.HTTPSOptions,
) *TLSBridge {
	return &TLSBridge{
		logger:    logger,
		desyncer:  desyncer,
		sniffer:   sniffer,
		httpsOpts: httpsOpts,
	}
}

// Tunnel creates a bi-directional tunnel between lConn and dst.
// It detects the first packet from lConn. If it's a ClientHello, it applies the desync strategy.
func (b *TLSBridge) Tunnel(
	ctx context.Context,
	lConn net.Conn,
	dst *netutil.Destination,
	rule *config.Rule,
) error {
	httpsOpts := b.httpsOpts
	if rule != nil {
		httpsOpts = httpsOpts.Merge(rule.HTTPS)
	}

	if b.sniffer != nil && ptr.FromPtr(httpsOpts.FakeCount) > 0 {
		b.sniffer.RegisterUntracked(dst.Addrs, dst.Port)
	}

	logger := logging.WithLocalScope(ctx, b.logger, "https")

	rConn, err := netutil.DialFastest(ctx, "tcp", dst.Addrs, dst.Port, dst.Timeout)
	if err != nil {
		return err
	}
	defer netutil.CloseConns(rConn)

	logger.Debug().Msgf("new remote conn -> %s", rConn.RemoteAddr())

	// Read the first message from the client (expected to be ClientHello)
	tlsMsg, err := proto.ReadTLSMessage(lConn)
	if err != nil {
		logger.Trace().Err(err).Msgf("failed to read first message from client")
		return nil // Client might have closed connection or sent garbage
	}

	logger.Debug().
		Int("len", tlsMsg.Len()).
		Msgf("client hello received <- %s", lConn.RemoteAddr())

	if !tlsMsg.IsClientHello() {
		logger.Trace().Int("len", tlsMsg.Len()).Msg("not a client hello. aborting")
		return nil
	}

	// Send ClientHello to the remote server (with desync if configured)
	n, err := b.sendClientHello(ctx, rConn, tlsMsg, httpsOpts)
	if err != nil {
		return fmt.Errorf("failed to send client hello: %w", err)
	}

	logger.Debug().
		Int("len", n).
		Msgf("sent client hello -> %s", rConn.RemoteAddr())

	// Start bi-directional tunneling
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

func (b *TLSBridge) sendClientHello(
	ctx context.Context,
	conn net.Conn,
	msg *proto.TLSMessage,
	httpsOpts *config.HTTPSOptions,
) (int, error) {
	logger := logging.WithLocalScope(ctx, b.logger, "client_hello")
	return b.desyncer.Send(ctx, logger, conn, msg, httpsOpts)
}
