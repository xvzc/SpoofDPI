package system

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/jackpal/gateway" // Import gateway
	"github.com/rs/zerolog"
)

// ResolveGatewayMACAddr finds the default gateway's IP and then resolves its MAC address
// by actively sending an ARP request.
func ResolveGatewayMACAddr(
	logger zerolog.Logger,
	handle *pcap.Handle,
	iface *net.Interface,
	srcIP net.IP, // [MODIFIED] srcIP is now a parameter
) (net.HardwareAddr, error) {
	// 1. Find Gateway (Router) IP
	gatewayIP, err := gateway.DiscoverGateway()
	if err != nil {
		return nil, fmt.Errorf("could not discover gateway: %w", err)
	}
	logger.Info().Msgf("gateway ip; %s;", gatewayIP.String())

	// 2. Get our local MAC (srcIP is now provided)
	// [REMOVED] getInterfaceIPv4 call
	srcMAC := iface.HardwareAddr

	// 3. Craft ARP request packet (L2 + L3)
	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, // Broadcast
		EthernetType: layers.EthernetTypeARP,
	}
	arp := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6, // MAC
		ProtAddressSize:   4, // IPv4
		Operation:         layers.ARPRequest,
		SourceHwAddress:   []byte(srcMAC),
		SourceProtAddress: []byte(srcIP.To4()),                        // Use parameter
		DstHwAddress:      []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, // "Who has...?"
		DstProtAddress:    []byte(gatewayIP.To4()),                    // "...this IP?"
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	if err := gopacket.SerializeLayers(buf, opts, eth, arp); err != nil {
		return nil, fmt.Errorf("failed to serialize ARP request: %w", err)
	}

	// 4. Set a BPF filter to capture *only* the ARP reply we care about
	filterStr := fmt.Sprintf(
		"arp and ether dst %s and src host %s", // 👈 (수정됨)
		srcMAC.String(),
		gatewayIP.String(),
	)
	if err := handle.SetBPFFilter(filterStr); err != nil {
		return nil, fmt.Errorf("failed to set ARP BPF filter: %w", err)
	}
	defer func() { _ = handle.SetBPFFilter("") }()

	// 5. Send the ARP request
	if err := handle.WritePacketData(buf.Bytes()); err != nil {
		return nil, fmt.Errorf("failed to send ARP request: %w", err)
	}

	// 6. Listen for the reply
	logger.Info().Msgf("arp request sent; %d bytes;", len(buf.Bytes()))
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	timeout := time.After(3 * time.Second) // 3-second timeout
	for {
		select {
		case packet := <-packetSource.Packets():
			if arpLayer := packet.Layer(layers.LayerTypeARP); arpLayer != nil {
				arpReply, _ := arpLayer.(*layers.ARP)
				// Check if it's a reply (Operation=2) and from the IP we expect
				if arpReply.Operation == layers.ARPReply &&
					net.IP(arpReply.SourceProtAddress).Equal(gatewayIP) {
					logger.Info().Msgf("arp reply received; %d bytes;", len(packet.Data()))

					return net.HardwareAddr(arpReply.SourceHwAddress), nil
				}
			}
		case <-timeout:
			return nil, fmt.Errorf("ARP request for %s timed out", gatewayIP)
		}
	}
}

// GetInterfaceIPv4 finds the first valid (non-loopback) IPv4 address
// on a given interface.
// [MODIFIED] This function is now public.
func GetInterfaceIPv4(iface *net.Interface) (net.IP, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if ip := ipnet.IP.To4(); ip != nil {
				if !ip.IsLoopback() {
					return ip, nil
				}
			}
		}
	}
	return nil, errors.New("no non-loopback IPv4 address found on interface")
}
