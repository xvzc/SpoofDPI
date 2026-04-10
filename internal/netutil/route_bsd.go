//go:build darwin || freebsd

package netutil

import (
	"github.com/jackpal/gateway"
)

// getDefaultGateway finds the default gateway on macOS/BSD
func getDefaultGateway() (string, error) {
	ip, err := gateway.DiscoverGateway()
	if err != nil {
		return "", err
	}
	return ip.String(), nil
}
