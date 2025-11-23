//go:build linux

package packet

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"
	"unsafe"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"golang.org/x/sys/unix"
)

// LinuxPcapHandle uses standard syscalls (via x/sys/unix) to capture/inject packets.
// This ensures compatibility across all Linux architectures (including 386/MIPS).
type LinuxPcapHandle struct {
	fd      int
	ifIndex int
	buf     []byte
}

// NewPcapHandle opens a raw socket using unix package.
func NewPcapHandle(iface *net.Interface) (Handle, error) {
	// 1. Protocol Setup (Network Byte Order)
	proto := htons(unix.ETH_P_ALL)

	// 2. Open Raw Socket (AF_PACKET, SOCK_RAW)
	// unix.Socket handles architecture differences (like socketcall on 386)
	fd, err := unix.Socket(unix.AF_PACKET, unix.SOCK_RAW, int(proto))
	if err != nil {
		return nil, fmt.Errorf("failed to open raw socket: %w", err)
	}

	// 3. Bind to Interface
	// Handle "any" interface logic (index 0)
	bindIndex := 0
	if iface != nil && iface.Name != "any" {
		bindIndex = iface.Index
	}

	sll := &unix.SockaddrLinklayer{
		Protocol: proto,
		Ifindex:  bindIndex,
	}

	if err := unix.Bind(fd, sll); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("failed to bind raw socket: %w", err)
	}

	h := &LinuxPcapHandle{
		fd:      fd,
		ifIndex: bindIndex,
		buf:     make([]byte, 65535),
	}

	// 4. Enable Promiscuous Mode
	// Only needed if binding to a specific interface
	if bindIndex != 0 {
		if err := h.SetPromiscuous(true); err != nil {
			fmt.Printf("warning: failed to set promiscuous mode: %v\n", err)
		}
	}

	return h, nil
}

// ReadPacketData reads using unix.Recvfrom
func (h *LinuxPcapHandle) ReadPacketData() ([]byte, gopacket.CaptureInfo, error) {
	n, _, err := unix.Recvfrom(h.fd, h.buf, 0)
	if err != nil {
		return nil, gopacket.CaptureInfo{}, err
	}

	data := make([]byte, n)
	copy(data, h.buf[:n])

	ci := gopacket.CaptureInfo{
		Timestamp:     time.Now(),
		CaptureLength: n,
		Length:        n,
	}

	return data, ci, nil
}

// WritePacketData injects packet using unix.Sendto
func (h *LinuxPcapHandle) WritePacketData(data []byte) error {
	addr := &unix.SockaddrLinklayer{
		Ifindex: h.ifIndex,
	}
	return unix.Sendto(h.fd, data, 0, addr)
}

// LinkType logic for handling "any" interface (Linux SLL) vs Ethernet
func (h *LinuxPcapHandle) LinkType() layers.LinkType {
	if h.ifIndex == 0 {
		// "any" interface uses Linux SLL (Cooked Mode)
		return layers.LinkTypeLinuxSLL
	}
	return layers.LinkTypeEthernet
}

func (h *LinuxPcapHandle) Close() {
	_ = unix.Close(h.fd)
}

// SetBPFRawInstructionFilter attaches BPF using unix helper
func (h *LinuxPcapHandle) SetBPFRawInstructionFilter(raw []BPFInstruction) error {
	filter := make([]unix.SockFilter, len(raw))
	for i, r := range raw {
		filter[i] = unix.SockFilter{
			Code: r.Op,
			Jt:   r.Jt,
			Jf:   r.Jf,
			K:    r.K,
		}
	}

	fprog := &unix.SockFprog{
		Len:    uint16(len(filter)),
		Filter: &filter[0],
	}

	// unix package handles the syscall number correctly for all archs
	if err := unix.SetsockoptSockFprog(h.fd, unix.SOL_SOCKET, unix.SO_ATTACH_FILTER, fprog); err != nil {
		return fmt.Errorf("failed to attach BPF filter: %w", err)
	}

	return nil
}

// ClearBPF detaches the filter
func (h *LinuxPcapHandle) ClearBPF() error {
	// Dummy value 0 is sufficient
	return unix.SetsockoptInt(h.fd, unix.SOL_SOCKET, unix.SO_DETACH_FILTER, 0)
}

// SetPromiscuous uses unix helper for PacketMreq
func (h *LinuxPcapHandle) SetPromiscuous(enable bool) error {
	mreq := &unix.PacketMreq{
		Ifindex: int32(h.ifIndex),
		Type:    unix.PACKET_MR_PROMISC,
	}

	opt := unix.PACKET_ADD_MEMBERSHIP
	if !enable {
		opt = unix.PACKET_DROP_MEMBERSHIP
	}

	return unix.SetsockoptPacketMreq(h.fd, unix.SOL_PACKET, opt, mreq)
}

// --- Endian Utils ---

func determineNativeEndian() binary.ByteOrder {
	buf := [2]byte{}
	*(*uint16)(unsafe.Pointer(&buf[0])) = uint16(0xABCD)
	switch buf {
	case [2]byte{0xCD, 0xAB}:
		return binary.LittleEndian
	case [2]byte{0xAB, 0xCD}:
		return binary.BigEndian
	default:
		panic("could not determine native endianness")
	}
}

func htons(v uint16) uint16 {
	if determineNativeEndian() == binary.LittleEndian {
		return (v << 8) | (v >> 8)
	}
	return v
}
