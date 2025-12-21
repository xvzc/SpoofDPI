package packet

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/cache"
	"github.com/xvzc/SpoofDPI/internal/logging"
	"github.com/xvzc/SpoofDPI/internal/session"
)

var _ Sniffer = (*TCPSniffer)(nil)

type TCPSniffer struct {
	logger zerolog.Logger

	nhopCache  cache.Cache
	defaultTTL uint8

	handle Handle
}

func NewTCPSniffer(
	logger zerolog.Logger,
	cache cache.Cache,
	handle Handle,
	defaultTTL uint8,
) *TCPSniffer {
	return &TCPSniffer{
		logger:     logger,
		nhopCache:  cache,
		handle:     handle,
		defaultTTL: defaultTTL,
	}
}

// --- HopTracker Methods ---

func (ts *TCPSniffer) Cache() cache.Cache {
	return ts.nhopCache
}

// StartCapturing begins monitoring for SYN/ACK packets in a background goroutine.
func (ts *TCPSniffer) StartCapturing() {
	// Create a new packet source from the handle.
	packetSource := gopacket.NewPacketSource(ts.handle, ts.handle.LinkType())
	packets := packetSource.Packets()
	// _ = ht.handle.SetBPFRawInstructionFilter(generateSynAckFilter())
	_ = ts.handle.ClearBPF()
	_ = ts.handle.SetBPFRawInstructionFilter(generateSynAckFilter())

	// Start a dedicated goroutine to process incoming packets.
	go func() {
		// Create a base context for this goroutine.
		ctx := session.WithNewTraceID(context.Background())
		for packet := range packets {
			ts.processPacket(ctx, packet)
		}
	}()
}

// RegisterUntracked registers new IP addresses for tracking.
// Addresses that are already being tracked are ignored.
func (ts *TCPSniffer) RegisterUntracked(addrs []net.IPAddr, port int) {
	portStr := strconv.Itoa(port)
	for _, v := range addrs {
		ts.nhopCache.Set(
			v.String()+":"+portStr,
			ts.defaultTTL,
			cache.Options().WithSkipExisting(true),
		)
	}
}

// GetOptimalTTL retrieves the estimated hop count for a given key from the cache.
// It returns the hop count and true if found, or 0 and false if not found.
func (ts *TCPSniffer) GetOptimalTTL(key string) uint8 {
	hopCount := uint8(255)
	if oTTL, ok := ts.nhopCache.Get(key); ok {
		hopCount = oTTL.(uint8)
	}

	return max(hopCount, 2) - 1
}

// processPacket analyzes a single packet to find SYN/ACKs and store hop counts.
func (ts *TCPSniffer) processPacket(ctx context.Context, p gopacket.Packet) {
	logger := logging.WithLocalScope(ctx, ts.logger, "sniff")

	tcpLayer := p.Layer(layers.LayerTypeTCP)
	if tcpLayer == nil {
		// log.Trace().Msgf("no tcp: %s", p.String())
		return
	}

	tcp, _ := tcpLayer.(*layers.TCP)
	if !tcp.SYN || !tcp.ACK {
		// log.Trace().Msgf("invalid packet: %s", p.String())
		return
	}

	// Check for a TCP layer
	var srcIP string
	var ttlLeft uint8

	// Handle IPv4
	if ipLayer := p.Layer(layers.LayerTypeIPv4); ipLayer != nil {
		ip, _ := ipLayer.(*layers.IPv4)
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

	// Create the cache key: ServerIP:ServerPort
	// (The source of the SYN/ACK is the server)
	key := fmt.Sprintf("%s:%d", srcIP, tcp.SrcPort)
	// Calculate hop count from the TTL
	nhops := estimateHops(ttlLeft)
	ok := ts.nhopCache.Set(key, nhops, nil)
	if ok {
		logger.Trace().
			Str("host_info", key).
			Uint8("nhops", nhops).
			Uint8("ttlLeft", ttlLeft).
			Msgf("received syn+ack")
	}
}

// estimateHops estimates the number of hops based on TTL.
// This logic is based on the hop counting mechanism from GoodbyeDPI.
// It returns 0 if the TTL is not recognizable.
func estimateHops(ttlLeft uint8) uint8 {
	// Unrecognizable initial TTL
	estimatedInitialHops := uint8(255)
	switch {
	case ttlLeft <= 64:
		estimatedInitialHops = 64
	case ttlLeft <= 128:
		estimatedInitialHops = 128
	default:
		estimatedInitialHops = 255
	}

	return estimatedInitialHops - ttlLeft
}

// GenerateSynAckFilter creates a BPF program for "ip and tcp and (tcp[13] & 18 == 18)".
// This captures only TCP SYN-ACK packets (IPv4).
func generateSynAckFilter() []BPFInstruction {
	instructions := []BPFInstruction{
		// 1. Check EtherType == IPv4 (0x0800)
		{Op: 0x28, Jt: 0, Jf: 0, K: 12},
		{Op: 0x15, Jt: 0, Jf: 8, K: 0x0800},

		// 2. Check Protocol == TCP (6)
		{Op: 0x30, Jt: 0, Jf: 0, K: 23},
		{Op: 0x15, Jt: 0, Jf: 6, K: 6},

		// 3. Check Fragmentation
		{Op: 0x28, Jt: 0, Jf: 0, K: 20},
		{Op: 0x45, Jt: 4, Jf: 0, K: 0x1fff},

		// 4. Find TCP Header Start (IP Header Length to X)
		// Loads byte at offset 14 (IP Header Start), gets IHL, multiplies by 4, stores in X.
		{Op: 0xb1, Jt: 0, Jf: 0, K: 14},

		// 5. Check TCP Flags (SYN+ACK)
		// We want to load: Ethernet(14) + IP_Len(X) + TCP_Flags(13)
		// Instruction is: Load [X + K]
		// So K must be 14 + 13 = 27.

		// [FIX] K was 13, changed to 27
		{Op: 0x50, Jt: 0, Jf: 0, K: 27},

		// Bitwise AND with 18 (SYN=2 | ACK=16)
		{Op: 0x54, Jt: 0, Jf: 0, K: 18},

		// Compare Result == 18
		{Op: 0x15, Jt: 0, Jf: 1, K: 18},

		// 6. Capture
		{Op: 0x6, Jt: 0, Jf: 0, K: 0x00040000},
		{Op: 0x6, Jt: 0, Jf: 0, K: 0x00000000},
	}

	return instructions
}
