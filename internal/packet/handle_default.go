//go:build !linux

package packet

import (
	"net"

	"github.com/google/gopacket/pcap"
)

var _ Handle = (*DefaultPcapHandle)(nil)

type DefaultPcapHandle struct {
	*pcap.Handle
}

func NewPcapHandle(iface *net.Interface) (Handle, error) {
	iHandle, err := pcap.NewInactiveHandle(iface.Name)
	if err != nil {
		return nil, err
	}

	// max bytes per packet to capture
	err = iHandle.SetSnapLen(3200)
	if err != nil {
		return nil, err
	}

	// in immediate mode, packets are delivered to the application
	// as soon as they arrive. In other words, this overrides SetTimeout.
	err = iHandle.SetImmediateMode(true)
	if err != nil {
		return nil, err
	}

	// create a pcap handle
	handle, err := iHandle.Activate()
	if err != nil {
		return nil, err
	}

	// activation successful, nil the inactive handle so defer doesn't close it
	iHandle = nil

	return &DefaultPcapHandle{handle}, err
}

func (h *DefaultPcapHandle) ClearBPF() error {
	return h.SetBPFFilter("")
}

func (h *DefaultPcapHandle) SetBPFRawInstructionFilter(
	inst []BPFInstruction,
) error {
	var converted []pcap.BPFInstruction
	for _, v := range inst {
		converted = append(converted, pcap.BPFInstruction{
			Code: v.Op, Jt: v.Jt, Jf: v.Jf, K: v.K,
		})
	}

	return h.SetBPFInstructionFilter(converted)
}
