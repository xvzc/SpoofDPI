package tun

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/desync"
	"github.com/xvzc/SpoofDPI/internal/logging"
	"github.com/xvzc/SpoofDPI/internal/matcher"
	"github.com/xvzc/SpoofDPI/internal/netutil"
	"github.com/xvzc/SpoofDPI/internal/packet"
	"github.com/xvzc/SpoofDPI/internal/proto"
)

type TCPHandler struct {
	logger           zerolog.Logger
	domainMatcher    matcher.RuleMatcher // For TLS domain matching only
	defaultHTTPSOpts *config.HTTPSOptions
	defaultConnOpts  *config.ConnOptions
	desyncer         *desync.TLSDesyncer
	sniffer          packet.Sniffer // For TTL tracking
	iface            string
	gateway          string
}

func NewTCPHandler(
	logger zerolog.Logger,
	domainMatcher matcher.RuleMatcher,
	defaultHTTPSOpts *config.HTTPSOptions,
	defaultConnOpts *config.ConnOptions,
	desyncer *desync.TLSDesyncer,
	sniffer packet.Sniffer,
	iface string,
	gateway string,
) *TCPHandler {
	return &TCPHandler{
		logger:           logger,
		domainMatcher:    domainMatcher,
		defaultHTTPSOpts: defaultHTTPSOpts,
		defaultConnOpts:  defaultConnOpts,
		desyncer:         desyncer,
		sniffer:          sniffer,
		iface:            iface,
		gateway:          gateway,
	}
}

func (h *TCPHandler) Handle(ctx context.Context, lConn net.Conn, rule *config.Rule) {
	logger := logging.WithLocalScope(ctx, h.logger, "tcp")

	defer netutil.CloseConns(lConn)

	// Set a read deadline for the first byte to avoid hanging indefinitely
	_ = lConn.SetReadDeadline(time.Now().Add(1 * time.Second))

	lBufferedConn := netutil.NewBufferedConn(lConn)
	buf, err := lBufferedConn.Peek(1)
	if err != nil {
		return
	}

	// Reset deadline
	_ = lConn.SetReadDeadline(time.Time{})

	// Parse destination from local address (which is the original destination in TUN)
	host, portStr, err := net.SplitHostPort(lConn.LocalAddr().String())
	if err != nil {
		return
	}
	port, _ := strconv.Atoi(portStr)

	ip := net.ParseIP(host)
	var iface *net.Interface
	if h.iface != "" {
		iface, _ = net.InterfaceByName(h.iface)
		logger.Debug().Str("iface", h.iface).Msg("using interface for dial")
	} else {
		logger.Debug().Msg("no interface specified for dial")
	}

	dst := &netutil.Destination{
		Domain:  host,
		Port:    port,
		Addrs:   []net.IP{},
		Iface:   iface,
		Gateway: h.gateway,
	}
	if h.defaultConnOpts != nil && h.defaultConnOpts.TCPTimeout != nil {
		dst.Timeout = *h.defaultConnOpts.TCPTimeout
	}
	if ip != nil {
		dst.Addrs = append(dst.Addrs, ip)
	}

	// Check if it's a TLS Handshake (Content Type 0x16)
	if buf[0] == 0x16 {
		logger.Debug().Msg("detected tls handshake")
		if err := h.handleTLS(ctx, logger, lBufferedConn, dst, rule); err != nil {
			logger.Debug().Err(err).Msg("tls handler failed")
		}
		return
	}

	// Handle as plain TCP
	rConn, err := netutil.DialFastest(ctx, "tcp", dst)
	if err != nil {
		logger.Error().Msgf("failed to dial %v", err)
		return
	}

	logger.Debug().Msgf("new remote conn -> %s", rConn.RemoteAddr())

	resCh := make(chan netutil.TransferResult, 2)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	startedAt := time.Now()
	go netutil.TunnelConns(ctx, resCh, lBufferedConn, rConn, netutil.TunnelDirOut)
	go netutil.TunnelConns(ctx, resCh, rConn, lBufferedConn, netutil.TunnelDirIn)

	err = netutil.WaitAndLogTunnel(
		ctx,
		logger,
		resCh,
		startedAt,
		netutil.DescribeRoute(lConn, rConn),
		nil,
	)
	if err != nil {
		logger.Error().Err(err).Msg("error handling request")
	}
}

func (h *TCPHandler) handleTLS(
	ctx context.Context,
	logger zerolog.Logger,
	lConn net.Conn,
	dst *netutil.Destination,
	addrRule *config.Rule, // Rule matched by IP in server.go
) error {
	// Read ClientHello
	tlsMsg, err := proto.ReadTLSMessage(lConn)
	if err != nil {
		return err
	}

	if !tlsMsg.IsClientHello() {
		return fmt.Errorf("not a client hello")
	}

	// Extract SNI
	start, end, err := tlsMsg.ExtractSNIOffset()
	if err != nil {
		return fmt.Errorf("failed to extract sni: %w", err)
	}
	dst.Domain = string(tlsMsg.Raw()[start:end])

	logger.Trace().Str("value", dst.Domain).Msgf("extracted sni feild")

	// Match Rules
	httpsOpts := h.defaultHTTPSOpts.Clone()
	connOpts := h.defaultConnOpts.Clone()

	// First, apply IP-based rule if matched in server.go
	if addrRule != nil {
		logger.Trace().RawJSON("summary", addrRule.JSON()).Msg("addr match")
		httpsOpts = httpsOpts.Merge(addrRule.HTTPS)
		connOpts = connOpts.Merge(addrRule.Conn)
	}

	// Then, try domain-based matching (TLS-specific)
	if h.domainMatcher != nil {
		domainSelector := &matcher.Selector{
			Kind:   matcher.MatchKindDomain,
			Domain: &dst.Domain,
		}
		if domainRule := h.domainMatcher.Search(domainSelector); domainRule != nil {
			logger.Trace().RawJSON("summary", domainRule.JSON()).Msg("domain match")
			// Domain rule takes priority if it has higher priority
			finalRule := matcher.GetHigherPriorityRule(addrRule, domainRule)
			if finalRule == domainRule {
				httpsOpts = h.defaultHTTPSOpts.Clone().Merge(domainRule.HTTPS)
				connOpts = h.defaultConnOpts.Clone().Merge(domainRule.Conn)
			}
		}
	}

	dst.Timeout = *connOpts.TCPTimeout

	// Dial Remote
	if h.sniffer != nil {
		h.sniffer.RegisterUntracked(dst.Addrs)
	}
	rConn, err := netutil.DialFastest(ctx, "tcp", dst)
	if err != nil {
		return err
	}
	defer netutil.CloseConns(rConn)

	logger.Debug().
		Msgf("new remote conn (%s -> %s)", lConn.RemoteAddr(), rConn.RemoteAddr())

	// Send ClientHello with Desync
	if _, err := h.desyncer.Desync(ctx, logger, rConn, tlsMsg, httpsOpts); err != nil {
		return err
	}

	// Tunnel rest
	resCh := make(chan netutil.TransferResult, 2)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	startedAt := time.Now()
	go netutil.TunnelConns(ctx, resCh, lConn, rConn, netutil.TunnelDirOut)
	go netutil.TunnelConns(ctx, resCh, rConn, lConn, netutil.TunnelDirIn)

	return netutil.WaitAndLogTunnel(
		ctx,
		logger,
		resCh,
		startedAt,
		netutil.DescribeRoute(lConn, rConn),
		nil,
	)
}

func (h *TCPHandler) SetNetworkInfo(iface, gateway string) {
	h.iface = iface
	h.gateway = gateway
}
