//go:build linux

package netutil

import (
	"github.com/jackpal/gateway"
)

// getDefaultGateway finds the default gateway on Linux
func getDefaultGateway() (string, error) {
	ip, err := gateway.DiscoverGateway()
	if err != nil {
		return "", err
	}
	return ip.String(), nil
}
