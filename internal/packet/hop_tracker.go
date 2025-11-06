package packet

import (
	"context"
	"fmt"
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
	handle *pcap.Handle
	// The cache stores hop counts (uint8)
	cache  *datastruct.TTLCache[uint8]
	logger zerolog.Logger
}

// NewHopTracker creates a new HopTracker.
func NewHopTracker(
	handle *pcap.Handle,
	cache *datastruct.TTLCache[uint8],
	logger zerolog.Logger,
) *HopTracker {
	// Error checking for nil handle and cache has been removed
	// as per the request.

	return &HopTracker{
		handle: handle,
		cache:  cache,
		logger: logger,
	}
}

// StartCapturing begins monitoring for SYN/ACK packets in a background goroutine.
func (ht *HopTracker) StartCapturing() {
	// Create a new packet source from the handle.
	packetSource := gopacket.NewPacketSource(ht.handle, ht.handle.LinkType())
	packets := packetSource.Packets()
	ht.handle.SetBPFFilter("tcp and (tcp[13] & 18 = 18)")

	// Start a dedicated goroutine to process incoming packets.
	go func() {
		// Create a base context for this goroutine.
		ctx := appctx.WithNewTraceID(context.Background())
		for packet := range packets {
			ht.processPacket(ctx, packet)
		}
	}()
}

// GetHops retrieves the estimated hop count for a given key from the cache.
// It returns the hop count and true if found, or 0 and false if not found.
func (ht *HopTracker) GetHops(key string) (uint8, bool) {
	return ht.cache.Get(key)
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
			cacheKey := fmt.Sprintf("%s:%d", srcIP, tcp.SrcPort)

			// Calculate hop count from the TTL
			nhops := calculateHops(ttl)
			if nhops > 0 {
				// Store the hop count in the cache with a short TTL
				ht.cache.Set(cacheKey, nhops, 180*time.Second)
				logger.Debug().
					Msgf("detected SYN/ACK and stored hop count: %s, TTL: %d, Hops: %d",
						cacheKey, ttl, nhops,
					)
			} else {
				logger.Debug().
					Msgf("detected SYN/ACK, but could not estimate hop count: %s, TTL: %d",
						cacheKey, ttl,
					)
			}
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
