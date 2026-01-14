package socks5

import (
	"context"
	"net"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/logging"
	"github.com/xvzc/SpoofDPI/internal/netutil"
	"github.com/xvzc/SpoofDPI/internal/proto"
)

type BindHandler struct {
	logger zerolog.Logger
}

func NewBindHandler(logger zerolog.Logger) *BindHandler {
	return &BindHandler{
		logger: logger,
	}
}

func (h *BindHandler) Handle(
	ctx context.Context,
	conn net.Conn,
	req *proto.SOCKS5Request,
) error {
	logger := logging.WithLocalScope(ctx, h.logger, "bind")

	// 1. Listen on a random TCP port
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		logger.Error().Err(err).Msg("failed to create bind listener")
		_ = proto.SOCKS5FailureResponse().Write(conn)
		return err
	}
	defer func() { _ = listener.Close() }()

	logger.Debug().
		Str("addr", listener.Addr().String()).
		Str("network", listener.Addr().Network()).
		Msg("new listener")

	lAddr := listener.Addr().(*net.TCPAddr)

	// 2. First Reply: Send the address/port we are listening on
	if err := proto.SOCKS5SuccessResponse().Bind(lAddr.IP).Port(lAddr.Port).Write(conn); err != nil {
		logger.Error().Err(err).Msg("failed to write first bind reply")
		return err
	}

	logger.Debug().
		Str("bind_addr", lAddr.String()).
		Msg("waiting for incoming connection")

	// 3. Accept Incoming Connection
	// The client should now tell the application server to connect to lAddr.
	remoteConn, err := listener.Accept()
	if err != nil {
		logger.Error().Err(err).Msg("failed to accept incoming connection")
		_ = proto.SOCKS5FailureResponse().Write(conn)
		return err
	}
	defer netutil.CloseConns(remoteConn)

	rAddr := remoteConn.RemoteAddr().(*net.TCPAddr)

	logger.Debug().
		Str("remote_addr", rAddr.String()).
		Msg("accepted incoming connection")

	// 4. Second Reply: Send the address/port of the connecting host
	if err := proto.SOCKS5SuccessResponse().Bind(rAddr.IP).Port(rAddr.Port).Write(conn); err != nil {
		logger.Error().Err(err).Msg("failed to write second bind reply")
		return err
	}

	// 5. Start bi-directional tunneling
	resCh := make(chan netutil.TransferResult, 2)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	startedAt := time.Now()
	go netutil.TunnelConns(ctx, resCh, remoteConn, conn, netutil.TunnelDirOut)
	go netutil.TunnelConns(ctx, resCh, conn, remoteConn, netutil.TunnelDirIn)

	return netutil.WaitAndLogTunnel(
		ctx,
		logger,
		resCh,
		startedAt,
		netutil.DescribeRoute(conn, remoteConn),
		nil,
	)
}
