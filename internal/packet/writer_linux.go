//go:build linux

package packet

import (
	"encoding/binary"
	"fmt"
	"net"
	"syscall"
	"unsafe"

	"github.com/google/gopacket/pcap"
)

var _ PacketWriter = (*LinuxPacketWriter)(nil)

type LinuxPacketWriter struct {
	fd      int
	ifIndex int
}

func NewPacketWriter(
	handle *pcap.Handle,
	iface *net.Interface,
) (*LinuxPacketWriter, error) {
	// ETH_P_ALL (0x0003) tells the kernel "we want to handle all protocols"
	// Warning: htons is required because the kernel expects Network Byte Order (Big Endian)
	// for the protocol field in the socket call.
	proto := htons(syscall.ETH_P_ALL)

	// AF_PACKET works on all Linux architectures (x86, ARM, MIPS, etc.)
	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(proto))
	if err != nil {
		return nil, fmt.Errorf("failed to open raw socket: %w", err)
	}

	// Bind enables us to send packets specifically out of this interface
	sll := syscall.SockaddrLinklayer{
		Protocol: proto,
		Ifindex:  iface.Index,
	}

	if err := syscall.Bind(fd, &sll); err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("failed to bind raw socket: %w", err)
	}

	return &LinuxPacketWriter{
		fd:      fd,
		ifIndex: iface.Index,
	}, nil
}

func (w *LinuxPacketWriter) WritePacketData(data []byte) error {
	addr := syscall.SockaddrLinklayer{
		Ifindex: w.ifIndex,
	}

	return syscall.Sendto(w.fd, data, 0, &addr)
}

func (w *LinuxPacketWriter) Close() {
	syscall.Close(w.fd)
}

// htons converts host byte order to network byte order (Big Endian).
func htons(v uint16) uint16 {
	buf := [2]byte{}
	*(*uint16)(unsafe.Pointer(&buf[0])) = uint16(0xABCD)

	var endian binary.ByteOrder
	switch buf {
	case [2]byte{0xCD, 0xAB}:
		endian = binary.LittleEndian
	case [2]byte{0xAB, 0xCD}:
		endian = binary.BigEndian
	default:
		panic("Could not determine native endianness")
	}

	if endian == binary.LittleEndian {
		return (v<<8)&0xff00 | (v>>8)&0x00ff
	}

	return v // Already Big Endian
}
