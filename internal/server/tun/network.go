//go:build !darwin && !linux && !freebsd

package tun

func SetRoute(iface string, subnets []string) error {
	return nil
}

func UnsetRoute(iface string, subnets []string) error {
	return nil
}

func SetInterfaceAddress(iface string, local string, remote string) error {
	return nil
}

func UnsetGatewayRoute(gateway, iface string) error {
	return nil
}

func SetGatewayRoute(gateway, iface string) error {
	return nil
}
