//go:build !linux

package packet

import (
	"net"

	"github.com/google/gopacket/pcap"
)

var _ PacketWriter = (*DefaultPacketWriter)(nil)

type DefaultPacketWriter struct {
	*pcap.Handle
}

func NewPacketWriter(
	handle *pcap.Handle,
	iface *net.Interface,
) (*DefaultPacketWriter, error) {
	return &DefaultPacketWriter{
		handle,
	}, nil
}
