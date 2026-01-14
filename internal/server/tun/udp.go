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
	logger   zerolog.Logger
	udpOpts  *config.UDPOptions
	desyncer *desync.UDPDesyncer
	pool     *netutil.ConnPool
	iface    string
	gateway  string
}

func NewUDPHandler(
	logger zerolog.Logger,
	desyncer *desync.UDPDesyncer,
	udpOpts *config.UDPOptions,
	pool *netutil.ConnPool,
) *UDPHandler {
	return &UDPHandler{
		logger:   logger,
		desyncer: desyncer,
		udpOpts:  udpOpts,
		pool:     pool,
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
	opts := h.udpOpts.Clone()
	if rule != nil {
		logger.Trace().RawJSON("summary", rule.JSON()).Msg("match")
		opts = opts.Merge(rule.UDP)
	}

	// Key for connection pool
	key := lConn.RemoteAddr().String() + ">" + lConn.LocalAddr().String()

	// Dial remote connection
	rawConn, err := netutil.DialFastest(ctx, "udp", dst)
	if err != nil {
		return
	}

	// Add to pool (pool handles LRU eviction and deadline)
	rConn := h.pool.Add(key, rawConn)

	// Wrap lConn with TimeoutConn as well
	timeout := *opts.Timeout
	lConnWrapped := &netutil.TimeoutConn{Conn: lConn, Timeout: timeout}

	// Desync
	_, _ = h.desyncer.Desync(ctx, lConnWrapped, rConn, opts)

	logger.Debug().
		Msgf("new remote conn (%s -> %s)", lConn.RemoteAddr(), rConn.RemoteAddr())

	resCh := make(chan netutil.TransferResult, 2)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	startedAt := time.Now()
	go netutil.TunnelConns(ctx, resCh, lConnWrapped, rConn, netutil.TunnelDirOut)
	go netutil.TunnelConns(ctx, resCh, rConn, lConnWrapped, netutil.TunnelDirIn)

	err = netutil.WaitAndLogTunnel(
		ctx,
		logger,
		resCh,
		startedAt,
		netutil.DescribeRoute(lConnWrapped, rConn),
		nil,
	)
	if err != nil {
		logger.Error().Err(err).Msg("error handling request")
	}
}
