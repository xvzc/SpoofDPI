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
	"sync"

	"github.com/rs/zerolog"
	"github.com/samber/lo"
	"github.com/xvzc/spoofdpi/internal/config"
	"github.com/xvzc/spoofdpi/internal/logging"
	"github.com/xvzc/spoofdpi/internal/matcher"
	"github.com/xvzc/spoofdpi/internal/netutil"
	"github.com/xvzc/spoofdpi/internal/server"
	"github.com/xvzc/spoofdpi/internal/session"
	"golang.zx2c4.com/wireguard/tun"
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

	tunDevice tun.Device
	iface     string
	gateway   string
}

func NewTunServer(
	logger zerolog.Logger,
	config *config.Config,
	matcher matcher.RuleMatcher,
	tcpHandler *TCPHandler,
	udpHandler *UDPHandler,
	iface string,
	gateway string,
) server.Server {
	return &TunServer{
		logger:     logger,
		config:     config,
		matcher:    matcher,
		tcpHandler: tcpHandler,
		udpHandler: udpHandler,
		iface:      iface,
		gateway:    gateway,
	}
}

func (s *TunServer) ListenAndServe(
	appctx context.Context,
) error {
	logger := logging.WithLocalScope(appctx, s.logger, "tun")

	var err error
	s.tunDevice, err = newTunDevice()
	if err != nil {
		return fmt.Errorf("failed to create tun device: %w", err)
	}

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

	go func() {
		<-appctx.Done()
		_ = s.tunDevice.Close()
	}()

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
	go func() {
		go s.tunToStack(appctx, logger, ep)
		s.stackToTun(appctx, logger, ep)
	}()

	return nil
}

func (s *TunServer) SetNetworkConfig() (func() error, error) {
	if s.tunDevice == nil {
		return nil, fmt.Errorf("tun device not initialized")
	}

	tunName, err := s.tunDevice.Name()
	if err != nil {
		return nil, fmt.Errorf("failed to get tun device name: %w", err)
	}

	local, remote, err := netutil.FindSafeSubnet()
	if err != nil {
		return nil, fmt.Errorf("failed to find safe subnet: %w", err)
	}

	if err := SetInterfaceAddress(tunName, local, remote); err != nil {
		return nil, fmt.Errorf("failed to set interface address: %w", err)
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

	err = SetRoute(tunName, []string{networkAddr.String() + "/30"})
	if err != nil {
		return nil, fmt.Errorf("failed to set local route: %w", err)
	}

	// Add a host route to the gateway via the physical interface
	// This ensures spoofdpi's outbound traffic goes through en0, not utun8
	if err := SetGatewayRoute(s.gateway, s.iface); err != nil {
		s.logger.Error().Err(err).Msg("failed to set gateway route")
	}

	err = SetRoute(tunName, []string{"0.0.0.0/0"}) // Default Route
	if err != nil {
		return nil, fmt.Errorf("failed to set default route: %w", err)
	}

	unset := func() error {
		if s.tunDevice == nil {
			return nil
		}

		// Remove the gateway route
		if s.gateway != "" && s.iface != "" {
			if err := UnsetGatewayRoute(s.gateway, s.iface); err != nil {
				s.logger.Warn().Err(err).Msg("failed to unset gateway route")
			}
		}

		return UnsetRoute(tunName, []string{"0.0.0.0/0"}) // Default Route
	}

	return unset, nil
}

func (s *TunServer) Addr() string {
	if s.tunDevice != nil {
		if name, err := s.tunDevice.Name(); err == nil {
			return name
		}
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

func (s *TunServer) tunToStack(
	appctx context.Context,
	logger zerolog.Logger,
	ep *channel.Endpoint,
) {
	const (
		// readOffset is the headroom before each IP packet in the read buffer.
		// wireguard-go on Linux (IFF_VNET_HDR) writes the virtio-net header into
		// buf[offset-virtioNetHdrLen:offset], so offset must be >= 10.
		// On macOS the value just acts as padding; >= 4 is sufficient.
		// We use 10 so a single constant works on all platforms.
		readOffset = 10
		mtu        = 1500
	)

	// Batch size: on Linux with IFF_VNET_HDR, BatchSize() returns
	// conn.IdealBatchSize (typically 128). handleVirtioRead → gsoSplit writes
	// each GRO sub-segment into a separate bufs[i] slot. If len(bufs) < number
	// of segments, gsoSplit returns ErrTooManySegments. We must therefore
	// pre-allocate exactly BatchSize() buffers.
	batchSize := s.tunDevice.BatchSize()

	// Allocate all per-packet buffers from a single contiguous backing array to
	// keep allocations low.
	const bufSize = readOffset + mtu
	backing := make([]byte, batchSize*bufSize)
	bufs := make([][]byte, batchSize)
	sizes := make([]int, batchSize)
	for i := range bufs {
		bufs[i] = backing[i*bufSize : (i+1)*bufSize]
	}

	for {
		// Reset sizes before each Read; wireguard-go overwrites them with actual
		// packet lengths.
		for i := range sizes {
			sizes[i] = mtu
		}

		n, err := s.tunDevice.Read(bufs, sizes, readOffset)
		if err != nil {
			if errors.Is(err, fs.ErrClosed) || errors.Is(err, os.ErrClosed) {
				return
			}
			select {
			case <-appctx.Done():
				return
			default:
				if err != io.EOF {
					logger.Error().Err(err).Msg("failed to read from tun")
				}
				return
			}
		}

		// Process each packet returned by this Read call (n >= 1 on success).
		for i := range n {
			if sizes[i] < 1 {
				continue
			}

			packet := bufs[i][readOffset : readOffset+sizes[i]]

			if packet[0]>>4 != 4 {
				logger.Trace().Int("version", int(packet[0]>>4)).Msg("skipping non-ipv4 packet")
				continue
			}

			payload := buffer.MakeWithData(append([]byte(nil), packet...))
			pkt := stack.NewPacketBuffer(stack.PacketBufferOptions{Payload: payload})
			ep.InjectInbound(ipv4.ProtocolNumber, pkt)
			pkt.DecRef()
		}
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
	appctx context.Context,
	logger zerolog.Logger,
	ep *channel.Endpoint,
) {
	ch := make(chan struct{}, 1)
	n := &notifier{ch: ch}
	ep.AddNotify(n)

	// Pool of []byte slices to avoid per-packet allocation.
	// Each buffer holds a headroom prefix followed by the IP packet.
	//
	// The offset must be >= 10 (virtioNetHdrLen) on Linux because wireguard-go
	// enables IFF_VNET_HDR, and its Write() implementation does:
	//   offset -= virtioNetHdrLen  (i.e. offset -= 10)
	// to place the virtio-net header in buf[offset-10:offset] before writing.
	// With offset=4 this would compute a negative index, silently dropping every
	// packet written back to the TUN device (no response ever reaches the client).
	// On macOS the TUN device only needs offset >= 4 (AF-family header), so 10 is
	// safe on all platforms.
	const writeOffset = 10
	pool := &sync.Pool{
		New: func() any {
			b := make([]byte, writeOffset+1500)
			return &b
		},
	}

	for {
		select {
		case <-appctx.Done():
			return
		default:
		}

		pkt := ep.Read()
		if pkt == nil {
			select {
			case <-ch:
				continue
			case <-appctx.Done():
				return
			}
		}

		views := pkt.ToView().AsSlice()
		if len(views) > 0 {
			// wireguard-go Write(bufs, offset) writes the 4-byte AF family header into
			// buf[offset-4:offset] and reads the IP packet from buf[offset:].
			// We must therefore prepend 4 zero bytes so the IP payload starts at index 4.
			needed := writeOffset + len(views)
			bp := pool.Get().(*[]byte)
			if cap(*bp) < needed {
				*bp = make([]byte, needed)
			}
			buf := (*bp)[:needed]
			copy(buf[writeOffset:], views)
			_, _ = s.tunDevice.Write([][]byte{buf}, writeOffset)
			pool.Put(bp)
		}
		pkt.DecRef()
	}
}

func newTunDevice() (tun.Device, error) {
	return tun.CreateTUN("tun", 1500)
}
