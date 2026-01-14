//go:build !darwin

package tun

func SetRouting(iface string, subnets []string) error {
	return nil
}

func UnsetRouting(iface string, subnets []string) error {
	return nil
}

func SetInterfaceAddress(iface string, local string, remote string) error {
	return nil
}
