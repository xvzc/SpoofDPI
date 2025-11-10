package packet

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/appctx"
	"github.com/xvzc/SpoofDPI/internal/datastruct"
)

// HopTracker monitors a pcap handle to find SYN/ACK packets and
// stores their estimated hop count into a TTLCache.
type HopTracker struct {
	logger zerolog.Logger

	cache  *datastruct.TTLCache[uint8] // The cache stores hop counts (uint8)
	handle *pcap.Handle
}

// NewHopTracker creates a new HopTracker.
func NewHopTracker(
	logger zerolog.Logger,
	cache *datastruct.TTLCache[uint8],
	handle *pcap.Handle,
) *HopTracker {
	// Error checking for nil handle and cache has been removed
	// as per the request.

	return &HopTracker{
		logger: logger,
		cache:  cache,
		handle: handle,
	}
}

// StartCapturing begins monitoring for SYN/ACK packets in a background goroutine.
func (ht *HopTracker) StartCapturing() {
	// Create a new packet source from the handle.
	packetSource := gopacket.NewPacketSource(ht.handle, ht.handle.LinkType())
	packets := packetSource.Packets()
	_ = ht.handle.SetBPFFilter("tcp and (tcp[13] & 18 = 18)")

	// Start a dedicated goroutine to process incoming packets.
	go func() {
		// Create a base context for this goroutine.
		ctx := appctx.WithNewTraceID(context.Background())
		for packet := range packets {
			ht.processPacket(ctx, packet)
		}
	}()
}

// GetOptimalHops retrieves the estimated hop count for a given key from the cache.
// It returns the hop count and true if found, or 0 and false if not found.
func (ht *HopTracker) GetOptimalHops(key string) uint8 {
	nhops, ok := ht.cache.Get(key)
	if !ok || nhops < 1 {
		return math.MaxUint8 - 1
	}

	return nhops - 1
}

// processPacket analyzes a single packet to find SYN/ACKs and store hop counts.
func (ht *HopTracker) processPacket(ctx context.Context, p gopacket.Packet) {
	logger := ht.logger.With().Ctx(ctx).Logger()

	// Check for a TCP layer
	if tcpLayer := p.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		tcp, _ := tcpLayer.(*layers.TCP)

		// Check if both SYN and ACK flags are set (a SYN/ACK response)
		if tcp.SYN && tcp.ACK {
			var srcIP string
			var ttl uint8

			// Handle IPv4
			if ipLayer := p.Layer(layers.LayerTypeIPv4); ipLayer != nil {
				ip, _ := ipLayer.(*layers.IPv4)
				srcIP = ip.SrcIP.String()
				ttl = ip.TTL
			} else if ipLayer := p.Layer(layers.LayerTypeIPv6); ipLayer != nil {
				// Handle IPv6
				ip, _ := ipLayer.(*layers.IPv6)
				srcIP = ip.SrcIP.String()
				ttl = ip.HopLimit
			} else {
				return // No IP layer found
			}

			// Create the cache key: ServerIP:ServerPort
			// (The source of the SYN/ACK is the server)
			key := fmt.Sprintf("%s:%d", srcIP, tcp.SrcPort)

			// Calculate hop count from the TTL
			nhops := calculateHops(ttl)
			ht.cache.Set(key, nhops, 180*time.Second)
			logger.Trace().
				Msgf("received syn+ack; src=%s; ttl=%d; nhops=%d;",
					key, ttl, nhops,
				)
		}
	}
}

// calculateHops estimates the number of hops based on TTL.
// This logic is based on the hop counting mechanism from GoodbyeDPI.
// It returns 0 if the TTL is not recognizable.
func calculateHops(ttl uint8) uint8 {
	if ttl > 98 && ttl < 128 {
		// Likely Windows (Initial TTL 128)
		return 128 - ttl
	} else if ttl > 34 && ttl < 64 {
		// Likely Linux/MacOS (Initial TTL 64)
		return 64 - ttl
	}
	// Unrecognizable initial TTL
	return 0
}
