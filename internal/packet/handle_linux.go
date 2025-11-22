//go:build linux

package packet

import (
	"fmt"
	"net"
	"time"

	"github.com/google/gopacket/afpacket"
	"github.com/google/gopacket/layers"
	"golang.org/x/net/bpf"
)

var _ Handle = (*LinuxPcapHandle)(nil)

type LinuxPcapHandle struct {
	*afpacket.TPacket
}

func NewPcapHandle(iface *net.Interface) (Handle, error) {
	// afpacket.NewTPacket opens a raw socket using memory mapping (Zero Copy)
	// It's much faster than libpcap and is Pure Go.
	tp, err := afpacket.NewTPacket(
		afpacket.OptInterface(iface.Name),
		afpacket.OptFrameSize(4096),
		afpacket.OptBlockSize(4096),
		afpacket.OptNumBlocks(128),
		afpacket.OptPollTimeout(time.Duration(-1)), // Block forever
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create afpacket handle: %w", err)
	}

	return &LinuxPcapHandle{tp}, nil
}

func (h *LinuxPcapHandle) SetBPFRawInstructionFilter(inst []BPFInstruction) error {
	var converted []bpf.RawInstruction
	for _, v := range inst {
		converted = append(converted, bpf.RawInstruction{
			Op: v.Op, Jt: v.Jt, Jf: v.Jf, K: v.K,
		})
	}

	return h.SetBPF(converted)
}

func (h *LinuxPcapHandle) LinkType() layers.LinkType {
	return layers.LinkTypeEthernet
}
