package socks5

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/rs/zerolog"
	"github.com/samber/lo"
	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/desync"
	"github.com/xvzc/SpoofDPI/internal/logging"
	"github.com/xvzc/SpoofDPI/internal/netutil"
	"github.com/xvzc/SpoofDPI/internal/packet"
	"github.com/xvzc/SpoofDPI/internal/proto"
)

type ConnectHandler struct {
	logger           zerolog.Logger
	desyncer         *desync.TLSDesyncer
	sniffer          packet.Sniffer
	appOpts          *config.AppOptions
	defaultConnOpts  *config.ConnOptions
	defaultHTTPSOpts *config.HTTPSOptions
}

func NewConnectHandler(
	logger zerolog.Logger,
	desyncer *desync.TLSDesyncer,
	sniffer packet.Sniffer,
	appOpts *config.AppOptions,
	defaultConnOpts *config.ConnOptions,
	defaultHTTPSOpts *config.HTTPSOptions,
) *ConnectHandler {
	return &ConnectHandler{
		logger:           logger,
		desyncer:         desyncer,
		sniffer:          sniffer,
		appOpts:          appOpts,
		defaultConnOpts:  defaultConnOpts,
		defaultHTTPSOpts: defaultHTTPSOpts,
	}
}

func (h *ConnectHandler) Handle(
	ctx context.Context,
	lConn net.Conn,
	req *proto.SOCKS5Request,
	dst *netutil.Destination,
	rule *config.Rule,
) error {
	httpsOpts := h.defaultHTTPSOpts.Clone()
	connOpts := h.defaultConnOpts.Clone()
	if rule != nil {
		httpsOpts = httpsOpts.Merge(rule.HTTPS)
		connOpts = connOpts.Merge(rule.Conn)
	}

	logger := logging.WithLocalScope(ctx, h.logger, "connect")

	// 1. Validate Destination
	ok, err := netutil.ValidateDestination(dst.Addrs, dst.Port, h.appOpts.ListenAddr)
	if err != nil {
		logger.Debug().Err(err).Msg("error determining if valid destination")
		if !ok {
			_ = proto.SOCKS5FailureResponse().Write(lConn)
			return err
		}
	}

	// 2. Check if blocked
	if rule != nil && *rule.Block {
		logger.Debug().Msg("request is blocked by policy")
		_ = proto.SOCKS5FailureResponse().Write(lConn)
		return netutil.ErrBlocked
	}

	// 3. Send Success Response
	err = proto.SOCKS5SuccessResponse().Bind(net.IPv4zero).Port(0).Write(lConn)
	if err != nil {
		logger.Error().Err(err).Msg("failed to write socks5 success reply")
		return err
	}

	// logger := logging.WithLocalScope(ctx, h.logger, "connect(tcp)")
	dst.Timeout = *connOpts.TCPTimeout

	rConn, err := netutil.DialFastest(ctx, "tcp", dst)
	if err != nil {
		return err
	}
	defer netutil.CloseConns(rConn)

	logger.Debug().Msgf("new remote conn -> %s", rConn.RemoteAddr())

	// Wrap lConn with a buffered reader to peek for TLS
	bufConn := netutil.NewBufferedConn(lConn)

	// Peek first byte to check for TLS Handshake (0x16)
	// We try to peek 1 byte.
	b, err := bufConn.Peek(1)
	if err == nil && b[0] == byte(proto.TLSHandshake) { // 0x16

		if h.sniffer != nil && lo.FromPtr(httpsOpts.FakeCount) > 0 {
			h.sniffer.RegisterUntracked(dst.Addrs)
		}

		return h.handleHTTPS(ctx, bufConn, rConn, httpsOpts)
	}

	// If not TLS, fall back to pure TCP tunnel
	logger.Debug().Msg("not a tls handshake. fallback to pure tcp")

	resCh := make(chan netutil.TransferResult, 2)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	startedAt := time.Now()
	go netutil.TunnelConns(ctx, resCh, rConn, bufConn, netutil.TunnelDirOut)
	go netutil.TunnelConns(ctx, resCh, bufConn, rConn, netutil.TunnelDirIn)

	return netutil.WaitAndLogTunnel(
		ctx,
		logger,
		resCh,
		startedAt,
		netutil.DescribeRoute(bufConn, rConn),
		nil,
	)
}

func (h *ConnectHandler) handleHTTPS(
	ctx context.Context,
	lConn net.Conn, // This is expected to be the BufferedConn
	rConn net.Conn,
	opts *config.HTTPSOptions,
) error {
	logger := logging.WithLocalScope(ctx, h.logger, "connect(tls)")

	// Read the first message from the client (expected to be ClientHello)
	tlsMsg, err := proto.ReadTLSMessage(lConn)
	if err != nil {
		if err == io.EOF || err.Error() == "unexpected EOF" {
			return nil
		}
		logger.Trace().Err(err).Msgf("failed to read first message from client")
		return err
	}

	// It starts with 0x16, but is it a ClientHello?
	if !tlsMsg.IsClientHello() {
		logger.Debug().
			Int("len", tlsMsg.Len()).
			Msg("not a client hello. fallback to pure tcp")

		// Forward the initial bytes we read
		if _, err := rConn.Write(tlsMsg.Raw()); err != nil {
			return fmt.Errorf("failed to write initial bytes to remote: %w", err)
		}
	} else {
		logger.Debug().
			Int("len", tlsMsg.Len()).
			Msgf("client hello received <- %s", lConn.RemoteAddr())

		// Send ClientHello to the remote server (with desync if configured)
		n, err := h.sendClientHello(ctx, rConn, tlsMsg, opts)
		if err != nil {
			return fmt.Errorf("failed to send client hello: %w", err)
		}

		logger.Debug().
			Int("len", n).
			Msgf("sent client hello -> %s", rConn.RemoteAddr())
	}

	resCh := make(chan netutil.TransferResult, 2)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	startedAt := time.Now()
	go netutil.TunnelConns(ctx, resCh, rConn, lConn, netutil.TunnelDirOut)
	go netutil.TunnelConns(ctx, resCh, lConn, rConn, netutil.TunnelDirIn)

	return netutil.WaitAndLogTunnel(
		ctx,
		logger,
		resCh,
		startedAt,
		netutil.DescribeRoute(lConn, rConn),
		nil,
	)
}

func (h *ConnectHandler) sendClientHello(
	ctx context.Context,
	rConn net.Conn,
	msg *proto.TLSMessage,
	opts *config.HTTPSOptions,
) (int, error) {
	if lo.FromPtr(opts.Skip) {
		return rConn.Write(msg.Raw())
	}

	return h.desyncer.Desync(ctx, h.logger, rConn, msg, opts)
}
