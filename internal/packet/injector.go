package packet

import (
	"context"
	"errors"
	"math/rand/v2"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/rs/zerolog"
)

// Injector is capable of crafting and injecting L2-L7 packets
// by manually building headers, bypassing the OS network stack.
type Injector struct {
	logger zerolog.Logger

	handle Handle
	iface  *net.Interface

	gatewayMAC net.HardwareAddr // The MAC of the default gateway
}

// NewPacketInjector creates a new packet injector for a specific interface.
// It requires the pcap handle, the interface name, and the gateway's MAC address.
func NewPacketInjector(
	logger zerolog.Logger,
	handle Handle,
	iface *net.Interface,
	gatewayMAC net.HardwareAddr, // Gateway MAC is now injected
) (*Injector, error) {
	return &Injector{
		logger:     logger,
		handle:     handle,
		iface:      iface,
		gatewayMAC: gatewayMAC, // Store the injected MAC
	}, nil
}

// WriteCraftedPacket crafts and injects a full TCP packet from a payload.
// It uses the pre-configured gateway MAC address.
func (inj *Injector) WriteCraftedPacket(
	ctx context.Context,
	src *net.TCPAddr,
	dst *net.TCPAddr,
	ttl uint8,
	payload []byte,
) (int, error) {
	// set variables for src/dst
	srcMAC := inj.iface.HardwareAddr
	dstMAC := inj.gatewayMAC // Use the stored MAC
	srcIP := src.IP.To4()
	dstIP := dst.IP.To4()
	srcPort := src.Port
	dstPort := dst.Port

	if srcIP == nil || dstIP == nil {
		return 0, errors.New("`WriteCraftedPacket()` currently only supports IPv4")
	}

	packetLayers, err := inj.createLayers(
		srcMAC,
		dstMAC,
		srcIP,
		dstIP,
		srcPort,
		dstPort,
		ttl,
	)
	if err != nil {
		return 0, err
	}

	// serialize the packet L2(optional) + L3 + L4 + payload
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true, // Recalculate checksums
		FixLengths:       true, // Fix lengths
	}

	packetLayers = append(packetLayers, gopacket.Payload(payload))

	err = gopacket.SerializeLayers(buf, opts, packetLayers...)
	if err != nil {
		return 0, err
	}

	// inject the raw L2 packet
	if err := inj.handle.WritePacketData(buf.Bytes()); err != nil {
		return 0, err
	}

	return len(payload), nil
}

func (inj *Injector) createLayers(
	srcMAC net.HardwareAddr,
	dstMAC net.HardwareAddr,
	srcIP net.IP,
	dstIP net.IP,
	srcPort int,
	dstPort int,
	ttl uint8,
) ([]gopacket.SerializableLayer, error) {
	var packetLayers []gopacket.SerializableLayer

	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}
	packetLayers = append(packetLayers, eth)

	// define ip layer
	ipLayer := &layers.IPv4{
		Version:  4,
		TTL:      ttl,
		Protocol: layers.IPProtocolTCP,
		SrcIP:    srcIP,
		DstIP:    dstIP,
	}
	packetLayers = append(packetLayers, ipLayer)

	// define tcp layer
	tcpLayer := &layers.TCP{
		SrcPort: layers.TCPPort(srcPort), // Use a random high port
		DstPort: layers.TCPPort(dstPort),
		Seq:     rand.Uint32(), // A random sequence number
		PSH:     true,          // Push the payload
		ACK:     true,          // Assuming this is part of an established flow
		Ack:     rand.Uint32(),
		Window:  12345,
	}
	packetLayers = append(packetLayers, tcpLayer)

	if err := tcpLayer.SetNetworkLayerForChecksum(ipLayer); err != nil {
		return nil, err
	}

	return packetLayers, nil
}
