//go:build linux

package packet

import (
	"encoding/binary"
	"fmt"
	"net"
	"syscall"
	"time"
	"unsafe"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

const (
	solPacket           = 263 // SOL_PACKET
	packetAddMembership = 1   // PACKET_ADD_MEMBERSHIP
	packetMrPromisc     = 1   // PACKET_MR_PROMISC
	sockDetachFilter    = 27
)

// LinuxPcapHandle uses standard syscalls to capture and inject packets on Linux.
type LinuxPcapHandle struct {
	fd      int
	ifIndex int
	buf     []byte
}

// NewPcapHandle opens a raw socket using pure syscalls.
func NewPcapHandle(iface *net.Interface) (Handle, error) {
	// 1. Protocol Setup (Network Byte Order)
	proto := htons(syscall.ETH_P_ALL)

	// 2. Open Raw Socket
	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(proto))
	if err != nil {
		return nil, fmt.Errorf("failed to open raw socket: %w", err)
	}

	// 3. Bind to Interface
	sll := syscall.SockaddrLinklayer{
		Protocol: proto,
		Ifindex:  iface.Index,
	}
	if err := syscall.Bind(fd, &sll); err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("failed to bind raw socket: %w", err)
	}

	h := &LinuxPcapHandle{
		fd:      fd,
		ifIndex: iface.Index,
		buf:     make([]byte, 65535),
	}

	return h, nil
}

// ReadPacketData reads raw packet data.
func (h *LinuxPcapHandle) ReadPacketData() ([]byte, gopacket.CaptureInfo, error) {
	n, _, err := syscall.Recvfrom(h.fd, h.buf, 0)
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

func (h *LinuxPcapHandle) WritePacketData(data []byte) error {
	addr := syscall.SockaddrLinklayer{Ifindex: h.ifIndex}
	return syscall.Sendto(h.fd, data, 0, &addr)
}

func (h *LinuxPcapHandle) LinkType() layers.LinkType {
	return layers.LinkTypeEthernet
}

func (h *LinuxPcapHandle) Close() {
	syscall.Close(h.fd)
}

func (h *LinuxPcapHandle) SetBPFRawInstructionFilter(raw []BPFInstruction) error {
	_ = h.ClearBPF()
	filter := make([]syscall.SockFilter, len(raw))
	for i, r := range raw {
		filter[i] = syscall.SockFilter{Code: r.Op, Jt: r.Jt, Jf: r.Jf, K: r.K}
	}
	fprog := syscall.SockFprog{Len: uint16(len(filter)), Filter: &filter[0]}

	// SOL_SOCKET = 1, SO_ATTACH_FILTER = 26
	return setsockopt(h.fd, 1, 26, unsafe.Pointer(&fprog), unsafe.Sizeof(fprog))
}

// ClearBPF removes any attached BPF filter from the socket.
// This puts the socket back into "capture everything" mode.
func (h *LinuxPcapHandle) ClearBPF() error {
	// dummy value (not used by kernel for detach, but setsockopt expects an argument)
	dummy := 0

	// 필터 제거 요청
	return setsockopt(
		h.fd,
		1,
		sockDetachFilter,
		unsafe.Pointer(&dummy),
		unsafe.Sizeof(dummy),
	)
}

// setsockopt wrapper
func setsockopt(fd, level, opt int, val unsafe.Pointer, len uintptr) error {
	_, _, errno := syscall.Syscall6(
		syscall.SYS_SETSOCKOPT,
		uintptr(fd),
		uintptr(level),
		uintptr(opt),
		uintptr(val),
		len,
		0,
	)

	if errno != 0 {
		return errno
	}

	return nil
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
