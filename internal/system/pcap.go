package system

import (
	"net"

	"github.com/google/gopacket/pcap"
)

func CreatePcapHandle(iface *net.Interface) (*pcap.Handle, error) {
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

	return handle, err
}
