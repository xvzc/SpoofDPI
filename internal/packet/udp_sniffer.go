package packet

import (
	"context"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/cache"
	"github.com/xvzc/SpoofDPI/internal/logging"
)

var _ Sniffer = (*UDPSniffer)(nil)

type UDPSniffer struct {
	logger zerolog.Logger

	nhopCache  cache.Cache
	defaultTTL uint8

	handle Handle
}

func NewUDPSniffer(
	logger zerolog.Logger,
	cache cache.Cache,
	handle Handle,
	defaultTTL uint8,
) *UDPSniffer {
	return &UDPSniffer{
		logger:     logger,
		nhopCache:  cache,
		handle:     handle,
		defaultTTL: defaultTTL,
	}
}

// --- HopTracker Methods ---

func (us *UDPSniffer) Cache() cache.Cache {
	return us.nhopCache
}

// StartCapturing begins monitoring for UDP packets in a background goroutine.
func (us *UDPSniffer) StartCapturing() {
	// Create a new packet source from the handle.
	packetSource := gopacket.NewPacketSource(us.handle, us.handle.LinkType())
	packets := packetSource.Packets()

	_ = us.handle.ClearBPF()
	_ = us.handle.SetBPFRawInstructionFilter(generateUdpFilter(us.handle.LinkType()))

	// Start a dedicated goroutine to process incoming packets.
	go func() {
		for packet := range packets {
			us.processPacket(context.Background(), packet)
		}
	}()
}

// RegisterUntracked registers new IP addresses for tracking.
// Addresses that are already being tracked are ignored.
func (us *UDPSniffer) RegisterUntracked(addrs []net.IP) {
	for _, v := range addrs {
		us.nhopCache.Set(v.String(), us.defaultTTL, cache.Options().WithSkipExisting(true))
	}
}

// GetOptimalTTL retrieves the estimated hop count for a given key from the cache.
// It returns the hop count and true if found, or 0 and false if not found.
func (us *UDPSniffer) GetOptimalTTL(key string) uint8 {
	hopCount := uint8(255)
	if oTTL, ok := us.nhopCache.Get(key); ok {
		hopCount = oTTL.(uint8)
	}

	return max(hopCount, 2) - 1
}

// processPacket analyzes a single packet to store hop counts.
func (us *UDPSniffer) processPacket(ctx context.Context, p gopacket.Packet) {
	logger := logging.WithLocalScope(ctx, us.logger, "sniff")

	udpLayer := p.Layer(layers.LayerTypeUDP)
	if udpLayer == nil {
		return
	}

	var srcIP string
	var ttlLeft uint8

	// Handle IPv4
	if ipLayer := p.Layer(layers.LayerTypeIPv4); ipLayer != nil {
		ip, _ := ipLayer.(*layers.IPv4)

		// Skip packets from local/private IPs (outgoing packets)
		if isLocalIP(ip.SrcIP) {
			return
		}
		// Skip packets where dst is not local (outgoing packets including our fake packets)
		if !isLocalIP(ip.DstIP) {
			return
		}

		srcIP = ip.SrcIP.String()
		ttlLeft = ip.TTL
	} else if ipLayer := p.Layer(layers.LayerTypeIPv6); ipLayer != nil {
		// Handle IPv6
		ip, _ := ipLayer.(*layers.IPv6)
		srcIP = ip.SrcIP.String()
		ttlLeft = ip.HopLimit
	} else {
		return // No IP layer found
	}

	key := srcIP
	// Calculate hop count from the TTL
	nhops := estimateHops(ttlLeft)

	stored, exists := us.nhopCache.Get(key)

	if us.nhopCache.Set(key, nhops, nil) {
		if !exists || stored != nhops {
			logger.Trace().
				Str("from", key).
				Uint8("nhops", nhops).
				Uint8("ttlLeft", ttlLeft).
				Msgf("ttl(udp) update")
		}
	}
}

// GenerateUdpFilter creates a BPF program for "ip and udp".
// It supports Ethernet, Null (Loopback/VPN), and Raw IP link types.
func generateUdpFilter(linkType layers.LinkType) []BPFInstruction {
	var baseOffset uint32

	// Determine the offset where the IP header begins
	switch linkType {
	case layers.LinkTypeEthernet:
		baseOffset = 14
	case layers.LinkTypeNull, layers.LinkTypeLoop: // BSD Loopback / macOS utun
		baseOffset = 4
	case layers.LinkTypeRaw: // Linux TUN
		baseOffset = 0
	default:
		// Fallback to Ethernet or handle error if necessary
		baseOffset = 14
	}

	instructions := []BPFInstruction{}

	// 1. Protocol Verification (IPv4)
	if linkType == layers.LinkTypeEthernet {
		// Check EtherType == IPv4 (0x0800) at offset 12
		instructions = append(
			instructions,
			BPFInstruction{Op: 0x28, Jt: 0, Jf: 0, K: 12}, // Ldh [12]
			BPFInstruction{
				Op: 0x15,
				Jt: 0,
				Jf: 3,
				K:  0x0800,
			}, // Jeq 0x800, True, False(Skip to End)
		)
	} else {
		// Check IP Version == 4 at the base offset
		// Load byte at baseOffset, mask 0xF0, check if 0x40
		instructions = append(instructions,
			BPFInstruction{Op: 0x30, Jt: 0, Jf: 0, K: baseOffset}, // Ldb [baseOffset]
			BPFInstruction{Op: 0x54, Jt: 0, Jf: 0, K: 0xf0},       // And 0xf0
			BPFInstruction{Op: 0x15, Jt: 0, Jf: 3, K: 0x40},       // Jeq 0x40, True, False(Skip to End)
		)
	}

	// 2. Check Protocol == UDP (17)
	// Protocol field is at IP header + 9 bytes
	instructions = append(instructions,
		BPFInstruction{Op: 0x30, Jt: 0, Jf: 0, K: baseOffset + 9}, // Ldb [baseOffset + 9]
		BPFInstruction{Op: 0x15, Jt: 0, Jf: 1, K: 17},             // Jeq 17, True, False
	)

	// 3. Capture
	instructions = append(instructions,
		BPFInstruction{Op: 0x6, Jt: 0, Jf: 0, K: 0x00040000}, // Ret capture_len
		BPFInstruction{Op: 0x6, Jt: 0, Jf: 0, K: 0x00000000}, // Ret 0
	)

	return instructions
}
