package system

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/jackpal/gateway" // Import gateway
	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/packet"
)

func FindGatewayIPAddr() (net.IP, error) {
	return gateway.DiscoverGateway()
}

// ResolveGatewayMACAddr finds the default gateway's IP and then resolves its MAC address
// by actively sending an ARP request.
func ResolveGatewayMACAddr(
	logger zerolog.Logger,
	handle packet.Handle,
	gatewayIP net.IP,
	iface *net.Interface,
	srcIP net.IP, // [MODIFIED] srcIP is now a parameter
) (net.HardwareAddr, error) {
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
	filter, err := generateArpFilter(srcMAC, gatewayIP)
	if err != nil {
		return nil, fmt.Errorf("failed to generate BPF instructions: %w", err)
	}

	if err := handle.SetBPFRawInstructionFilter(filter); err != nil {
		return nil, fmt.Errorf("failed to set ARP BPF filter: %w", err)
	}
	// defer func() { _ = handle.SetBPFRawInstructionFilter([]packet.BPFInstruction{}) }()

	// 5. Send the ARP request
	if err := handle.WritePacketData(buf.Bytes()); err != nil {
		return nil, fmt.Errorf("failed to send ARP request: %w", err)
	}

	// 6. Listen for the reply
	logger.Trace().Int("len", len(buf.Bytes())).Msg("arp request sent")
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
					logger.Trace().Int("len", len(packet.Data())).Msg("arp reply received")

					return net.HardwareAddr(arpReply.SourceHwAddress), nil
				}
			}
		case <-timeout:
			return nil, fmt.Errorf("ARP request for %s timed out", gatewayIP)
		}
	}
}

func generateArpFilter(
	dstMAC net.HardwareAddr,
	srcIP net.IP,
) ([]packet.BPFInstruction, error) {
	if len(dstMAC) != 6 {
		return nil, fmt.Errorf("invalid MAC address length")
	}
	srcIP = srcIP.To4()
	if srcIP == nil {
		return nil, fmt.Errorf("invalid IPv4 address")
	}

	// 1. Convert MAC address (6 bytes) to integers: 4 bytes (High) + 2 bytes (Low) for comparison
	macHigh := binary.BigEndian.Uint32(dstMAC[0:4])
	macLow := binary.BigEndian.Uint16(dstMAC[4:6])

	// 2. Convert IP address (4 bytes) to an integer
	ipVal := binary.BigEndian.Uint32(srcIP)

	// BPF Instructions
	// Structure: {Op(Opcode), Jt(Jump True), Jf(Jump False), K(Value/Offset)}
	instructions := []packet.BPFInstruction{
		// -------------------------------------------------------
		// 1. Check if it is an ARP packet (EtherType == 0x0806)
		// -------------------------------------------------------
		// Load Absolute (Offset 12, Size 2 bytes - EtherType)
		{Op: 0x28, Jt: 0, Jf: 0, K: 12},
		// Jump If Equal (Val == 0x0806 ? Next : Fail)
		// Jf=7: Jump 7 instructions forward (to Fail)
		{Op: 0x15, Jt: 0, Jf: 7, K: 0x0806},

		// -------------------------------------------------------
		// 2. Check Ether Dst MAC (Offset 0, 6 bytes)
		// -------------------------------------------------------
		// BPF can compare up to 4 bytes at a time, so split the comparison into 4+2 bytes

		// [MAC High 4 bytes] Load Absolute (Offset 0, Size 4 bytes)
		{Op: 0x20, Jt: 0, Jf: 0, K: 0},
		// Compare with macHigh. Jf=5 (Fail)
		{Op: 0x15, Jt: 0, Jf: 5, K: macHigh},

		// [MAC Low 2 bytes] Load Absolute (Offset 4, Size 2 bytes)
		{Op: 0x28, Jt: 0, Jf: 0, K: 4},
		// Compare with macLow. Jf=3 (Fail)
		{Op: 0x15, Jt: 0, Jf: 3, K: uint32(macLow)},

		// -------------------------------------------------------
		// 3. Check ARP Sender IP (Offset 28, 4 bytes)
		// -------------------------------------------------------
		// Calculation: EtherHeader(14) + ARP HWType(2) + Proto(2) + HWLen(1) + ProtoLen(1) + Op(2) + SenderMAC(6)
		//             = Sender IP starts at the 28th byte

		// Load Absolute (Offset 28, Size 4 bytes)
		{Op: 0x20, Jt: 0, Jf: 0, K: 28},
		// Compare with ipVal. Jf=1 (Fail)
		{Op: 0x15, Jt: 0, Jf: 1, K: ipVal},

		// -------------------------------------------------------
		// 4. Return Result
		// -------------------------------------------------------
		// [Success] Ret 262144 (Capture full packet)
		{Op: 0x6, Jt: 0, Jf: 0, K: 0x00040000},
		// [Fail] Ret 0 (Drop packet)
		{Op: 0x6, Jt: 0, Jf: 0, K: 0x00000000},
	}

	return instructions, nil
}
