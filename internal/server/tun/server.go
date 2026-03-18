package tun

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"strconv"

	"github.com/rs/zerolog"
	"github.com/samber/lo"
	"github.com/songgao/water"
	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/logging"
	"github.com/xvzc/SpoofDPI/internal/matcher"
	"github.com/xvzc/SpoofDPI/internal/netutil"
	"github.com/xvzc/SpoofDPI/internal/server"
	"github.com/xvzc/SpoofDPI/internal/session"
	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
	"gvisor.dev/gvisor/pkg/waiter"
)

// Ensure tcpip is used to avoid "imported and not used" error
var _ tcpip.NetworkProtocolNumber = ipv4.ProtocolNumber

type TunServer struct {
	logger  zerolog.Logger
	config  *config.Config
	matcher matcher.RuleMatcher // For IP-based rule matching

	tcpHandler *TCPHandler
	udpHandler *UDPHandler

	iface          *water.Interface
	defaultIface   string
	defaultGateway string
}

func NewTunServer(
	logger zerolog.Logger,
	config *config.Config,
	matcher matcher.RuleMatcher,
	tcpHandler *TCPHandler,
	udpHandler *UDPHandler,
) server.Server {
	return &TunServer{
		logger:     logger,
		config:     config,
		matcher:    matcher,
		tcpHandler: tcpHandler,
		udpHandler: udpHandler,
	}
}

func (s *TunServer) Start(ctx context.Context, ready chan<- struct{}) error {
	iface, err := NewTunDevice()
	if err != nil {
		return fmt.Errorf("failed to create tun device: %w", err)
	}
	s.iface = iface

	if ready != nil {
		close(ready)
	}

	return s.handle(ctx, iface)
}

func (s *TunServer) Stop() error {
	if s.iface != nil {
		return s.iface.Close()
	}
	return nil
}

func (s *TunServer) SetNetworkConfig() error {
	if s.iface == nil {
		return fmt.Errorf("tun device not initialized")
	}

	// Find default interface and gateway before modifying routes
	defaultIface, defaultGateway, err := netutil.GetDefaultInterfaceAndGateway()
	if err != nil {
		return fmt.Errorf("failed to get default interface: %w", err)
	}
	s.logger.Info().
		Str("interface", defaultIface).
		Str("gateway", defaultGateway).
		Msg("determined default interface and gateway")
	s.defaultIface = defaultIface
	s.defaultGateway = defaultGateway

	// Update handlers with network info
	s.tcpHandler.SetNetworkInfo(defaultIface, defaultGateway)
	s.udpHandler.SetNetworkInfo(defaultIface, defaultGateway)

	local, remote, err := netutil.FindSafeSubnet()
	if err != nil {
		return fmt.Errorf("failed to find safe subnet: %w", err)
	}

	if err := SetInterfaceAddress(s.iface.Name(), local, remote); err != nil {
		return fmt.Errorf("failed to set interface address: %w", err)
	}

	// Add route for the TUN interface subnet to ensure packets can return
	// This is crucial for the TUN interface to receive packets destined for its own subnet
	// Calculate the network address for /30 subnet (e.g., 10.0.0.1 -> 10.0.0.0/30)
	localIP := net.ParseIP(local)
	networkAddr := net.IPv4(
		localIP[12],
		localIP[13],
		localIP[14],
		localIP[15]&0xFC,
	) // Mask with /30
	if err := SetRoute(s.iface.Name(), []string{networkAddr.String() + "/30"}); err != nil {
		return fmt.Errorf("failed to set local route: %w", err)
	}

	// Add a host route to the gateway via the physical interface
	// This ensures SpoofDPI's outbound traffic goes through en0, not utun8
	if err := SetGatewayRoute(defaultGateway, defaultIface); err != nil {
		s.logger.Warn().Err(err).Msg("failed to set gateway route")
	}

	return SetRoute(s.iface.Name(), []string{"0.0.0.0/0"}) // Default Route
}

func (s *TunServer) UnsetNetworkConfig() error {
	if s.iface == nil {
		return nil
	}

	// Remove the gateway route
	if s.defaultGateway != "" && s.defaultIface != "" {
		if err := UnsetGatewayRoute(s.defaultGateway, s.defaultIface); err != nil {
			s.logger.Warn().Err(err).Msg("failed to unset gateway route")
		}
	}

	return UnsetRoute(s.iface.Name(), []string{"0.0.0.0/0"}) // Default Route
}

func (s *TunServer) Addr() string {
	if s.iface != nil {
		return s.iface.Name()
	}
	return "tun"
}

// matchRuleByAddr extracts IP and port from net.Addr and performs rule matching
func (s *TunServer) matchRuleByAddr(addr net.Addr) *config.Rule {
	if s.matcher == nil {
		return nil
	}

	host, portStr, err := net.SplitHostPort(addr.String())
	if err != nil {
		return nil
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return nil
	}

	port, _ := strconv.Atoi(portStr)

	selector := &matcher.Selector{
		Kind: matcher.MatchKindAddr,
		IP:   lo.ToPtr(ip),
		Port: lo.ToPtr(uint16(port)),
	}

	return s.matcher.Search(selector)
}

func (s *TunServer) handle(ctx context.Context, iface *water.Interface) error {
	logger := logging.WithLocalScope(ctx, s.logger, "tun")

	// 1. Create gVisor stack
	stk := stack.New(stack.Options{
		NetworkProtocols: []stack.NetworkProtocolFactory{ipv4.NewProtocol},
		TransportProtocols: []stack.TransportProtocolFactory{
			tcp.NewProtocol,
			udp.NewProtocol,
		},
	})

	// 2. Create channel endpoint
	ep := channel.New(256, 1500, "")

	const nicID = 1
	if err := stk.CreateNIC(nicID, ep); err != nil {
		return fmt.Errorf("failed to create NIC: %v", err)
	}

	// 3. Enable Promiscuous mode & Spoofing
	stk.SetPromiscuousMode(nicID, true)
	stk.SetSpoofing(nicID, true)

	// 3.5. Add default route to the stack
	// Define a subnet that matches all IPv4 addresses (0.0.0.0/0)
	defaultSubnet, _ := tcpip.NewSubnet(
		tcpip.AddrFrom4([4]byte{0, 0, 0, 0}),
		tcpip.MaskFrom("\x00\x00\x00\x00"),
	)

	stk.SetRouteTable([]tcpip.Route{
		{
			Destination: defaultSubnet,
			NIC:         nicID,
		},
	})

	// 4. Register TCP Forwarder
	tcpFwd := tcp.NewForwarder(stk, 0, 65535, func(r *tcp.ForwarderRequest) {
		var wq waiter.Queue
		ep, err := r.CreateEndpoint(&wq)
		if err != nil {
			logger.Error().Msgf("failed to create endpoint: %v", err)
			r.Complete(true)
			return
		}
		r.Complete(false)

		conn := gonet.NewTCPConn(&wq, ep)

		// Match rule by IP before passing to handler
		rule := s.matchRuleByAddr(conn.LocalAddr())
		go s.tcpHandler.Handle(session.WithNewTraceID(context.Background()), conn, rule)
	})
	stk.SetTransportProtocolHandler(tcp.ProtocolNumber, tcpFwd.HandlePacket)

	// 5. Register UDP Forwarder
	udpFwd := udp.NewForwarder(stk, func(r *udp.ForwarderRequest) bool {
		var wq waiter.Queue
		ep, err := r.CreateEndpoint(&wq)
		if err != nil {
			logger.Error().Msgf("failed to create udp endpoint: %v", err)
			return true
		}

		conn := gonet.NewUDPConn(&wq, ep)

		// Match rule by IP before passing to handler
		rule := s.matchRuleByAddr(conn.LocalAddr())
		go s.udpHandler.Handle(session.WithNewTraceID(context.Background()), conn, rule)
		return true
	})
	stk.SetTransportProtocolHandler(udp.ProtocolNumber, udpFwd.HandlePacket)

	// 6. Start packet pump
	go s.tunToStack(ctx, logger, iface, ep)
	go s.stackToTun(ctx, logger, iface, ep)

	<-ctx.Done()
	return nil
}

func (s *TunServer) tunToStack(
	ctx context.Context,
	logger zerolog.Logger,
	iface *water.Interface,
	ep *channel.Endpoint,
) {
	buf := make([]byte, 2000)
	for {
		n, err := iface.Read(buf)
		if err != nil {
			if errors.Is(err, fs.ErrClosed) || errors.Is(err, os.ErrClosed) {
				return
			}

			select {
			case <-ctx.Done():
				return
			default:
				if err != io.EOF {
					logger.Error().Err(err).Msg("failed to read from tun")
				}
				return
			}
		}

		if n < 1 {
			continue
		}

		version := (buf[0] >> 4)
		if version != 4 {
			logger.Trace().Int("version", int(version)).Msg("skipping non-ipv4 packet")
			continue
		}

		// Parse source and destination IP for debugging
		// if n >= 20 {
		// 	srcIP := net.IP(buf[12:16])
		// 	dstIP := net.IP(buf[16:20])
		// 	protocol := buf[9]
		// 	logger.Trace().
		// 		Str("src", srcIP.String()).
		// 		Str("dst", dstIP.String()).
		// 		Uint8("proto", protocol).
		// 		Int("len", n).
		// 		Msg("injecting packet to stack")
		// }

		payload := buffer.MakeWithData(append([]byte(nil), buf[:n]...))

		pkt := stack.NewPacketBuffer(stack.PacketBufferOptions{
			Payload: payload,
		})
		ep.InjectInbound(ipv4.ProtocolNumber, pkt)
		pkt.DecRef()
	}
}

type notifier struct {
	ch chan<- struct{}
}

func (n *notifier) WriteNotify() {
	select {
	case n.ch <- struct{}{}:
	default:
	}
}

func (s *TunServer) stackToTun(
	ctx context.Context,
	logger zerolog.Logger,
	iface *water.Interface,
	ep *channel.Endpoint,
) {
	ch := make(chan struct{}, 1)
	n := &notifier{ch: ch}
	ep.AddNotify(n)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		pkt := ep.Read()
		if pkt == nil {
			select {
			case <-ch:
				continue
			case <-ctx.Done():
				return
			}
		}

		views := pkt.ToView().AsSlice()
		if len(views) > 0 {
			_, _ = iface.Write(views)
		}
		pkt.DecRef()
	}
}

func NewTunDevice() (*water.Interface, error) {
	config := water.Config{
		DeviceType: water.TUN,
	}
	return water.New(config)
}
