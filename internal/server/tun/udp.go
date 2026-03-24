package tun

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/desync"
	"github.com/xvzc/SpoofDPI/internal/logging"
	"github.com/xvzc/SpoofDPI/internal/netutil"
)

type UDPHandler struct {
	logger          zerolog.Logger
	defaultUDPOpts  *config.UDPOptions
	defaultConnOpts *config.ConnOptions
	desyncer        *desync.UDPDesyncer
	iface           string
	gateway         string
}

func NewUDPHandler(
	logger zerolog.Logger,
	desyncer *desync.UDPDesyncer,
	defaultUDPOpts *config.UDPOptions,
	defaultConnOpts *config.ConnOptions,
	iface string,
	gateway string,
) *UDPHandler {
	return &UDPHandler{
		logger:          logger,
		desyncer:        desyncer,
		defaultUDPOpts:  defaultUDPOpts,
		defaultConnOpts: defaultConnOpts,
		iface:           iface,
		gateway:         gateway,
	}
}

func (h *UDPHandler) SetNetworkInfo(iface, gateway string) {
	h.iface = iface
	h.gateway = gateway
}

func (h *UDPHandler) Handle(ctx context.Context, lConn net.Conn, rule *config.Rule) {
	logger := logging.WithLocalScope(ctx, h.logger, "udp")

	defer netutil.CloseConns(lConn)

	host, portStr, err := net.SplitHostPort(lConn.LocalAddr().String())
	if err != nil {
		return
	}
	port, _ := strconv.Atoi(portStr)

	var iface *net.Interface
	if h.iface != "" {
		iface, _ = net.InterfaceByName(h.iface)
	}

	dst := &netutil.Destination{
		Domain:  host,
		Port:    port,
		Iface:   iface,
		Gateway: h.gateway,
	}
	if ip := net.ParseIP(host); ip != nil {
		dst.Addrs = []net.IP{ip}
	}

	// Apply rule if matched in server.go
	udpOpts := h.defaultUDPOpts.Clone()
	connOpts := h.defaultConnOpts.Clone()
	if rule != nil {
		logger.Trace().RawJSON("summary", rule.JSON()).Msg("match")
		udpOpts = udpOpts.Merge(rule.UDP)
		connOpts = connOpts.Merge(rule.Conn)
	}

	// Dial remote connection
	rawConn, err := netutil.DialFastest(ctx, "udp", dst)
	if err != nil {
		logger.Error().Msgf("error dialing to %s", dst.String())
		return
	}

	timeout := *connOpts.UDPIdleTimeout

	// Wrap rConn with IdleTimeoutConn
	rConnWrapped := netutil.NewIdleTimeoutConn(rawConn, timeout)

	// Wrap lConn with IdleTimeoutConn as well
	lConnWrapped := netutil.NewIdleTimeoutConn(lConn, timeout)

	// Desync
	_, _ = h.desyncer.Desync(ctx, lConnWrapped, rConnWrapped, udpOpts)

	logger.Debug().
		Msgf("new remote conn (%s -> %s)", lConn.RemoteAddr(), rConnWrapped.RemoteAddr())

	resCh := make(chan netutil.TransferResult, 2)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	startedAt := time.Now()
	go netutil.TunnelConns(ctx, resCh, lConnWrapped, rConnWrapped, netutil.TunnelDirOut)
	go netutil.TunnelConns(ctx, resCh, rConnWrapped, lConnWrapped, netutil.TunnelDirIn)

	err = netutil.WaitAndLogTunnel(
		ctx,
		logger,
		resCh,
		startedAt,
		netutil.DescribeRoute(lConnWrapped, rConnWrapped),
		nil,
	)
	if err != nil {
		logger.Error().Err(err).Msg("error handling request")
	}
}
