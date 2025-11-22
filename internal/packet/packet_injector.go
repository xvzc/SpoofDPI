package packet

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/rs/zerolog"
)

// PacketInjector is capable of crafting and injecting L2-L7 packets
// by manually building headers, bypassing the OS network stack.
type PacketInjector struct {
	logger zerolog.Logger

	gatewayMAC net.HardwareAddr // The MAC of the default gateway
	handle     Handle
	iface      *net.Interface
}

// NewPacketInjector creates a new packet injector for a specific interface.
// It requires the pcap handle, the interface name, and the gateway's MAC address.
func NewPacketInjector(
	logger zerolog.Logger,
	gatewayMAC net.HardwareAddr, // Gateway MAC is now injected
	handle Handle,
	iface *net.Interface,
) (*PacketInjector, error) {
	return &PacketInjector{
		logger:     logger,
		gatewayMAC: gatewayMAC, // Store the injected MAC
		handle:     handle,
		iface:      iface,
	}, nil
}

// WriteCraftedPacket crafts and injects a full TCP packet from a payload.
// It uses the pre-configured gateway MAC address.
func (inj *PacketInjector) WriteCraftedPacket(
	ctx context.Context,
	src *net.TCPAddr,
	dst *net.TCPAddr,
	ttl uint8,
	payload []byte,
	repeat uint8,
) error {
	logger := inj.logger.With().Ctx(ctx).Logger()

	// set variables for src/dst
	srcMAC := inj.iface.HardwareAddr
	dstMAC := inj.gatewayMAC // Use the stored MAC
	srcIP := src.IP.To4()
	dstIP := dst.IP.To4()
	srcPort := src.Port
	dstPort := dst.Port

	if srcIP == nil || dstIP == nil {
		return errors.New("'InjectPakcet()' currently only supports IPv4")
	}

	totalSent := 0
	for range repeat {
		// define eth layer
		ethLayer := &layers.Ethernet{
			SrcMAC:       srcMAC,
			DstMAC:       dstMAC, // Use the stored MAC
			EthernetType: layers.EthernetTypeIPv4,
		}

		// define ip layer
		ipLayer := &layers.IPv4{
			Version:  4,
			TTL:      ttl,
			Protocol: layers.IPProtocolTCP,
			SrcIP:    srcIP,
			DstIP:    dstIP,
		}

		// define tcp layer
		tcpLayer := &layers.TCP{
			SrcPort: layers.TCPPort(srcPort), // Use a random high port
			DstPort: layers.TCPPort(dstPort),
			Seq:     uint32(rand.Int()), // A random sequence number
			PSH:     true,               // Push the payload
			ACK:     true,               // Assuming this is part of an established flow
			Ack:     uint32(rand.Int()),
			Window:  12345,
		}
		if err := tcpLayer.SetNetworkLayerForChecksum(ipLayer); err != nil {
			return fmt.Errorf("failed to set network layer for checksum: %w", err)
		}

		// serialize the packet (L2 + L3 + L4 + payload)
		buf := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{
			ComputeChecksums: true, // Recalculate checksums
			FixLengths:       true, // Fix lengths
		}

		err := gopacket.SerializeLayers(buf, opts,
			ethLayer,
			ipLayer,
			tcpLayer,
			gopacket.Payload(payload),
		)
		if err != nil {
			return fmt.Errorf("failed to serialize packet: %w", err)
		}

		// inject the raw L2 packet
		if err := inj.handle.WritePacketData(buf.Bytes()); err != nil {
			return fmt.Errorf("failed to inject packet: %w", err)
		}

		totalSent += len(payload)
	}

	logger.Debug().Msgf(
		"fake packets sent; dest=%s; ttl=%d; len=%d;", dstIP, ttl, totalSent,
	)

	return nil
}
