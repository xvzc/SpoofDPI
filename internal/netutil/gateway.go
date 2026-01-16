//go:build !darwin && !freebsd && !linux

package netutil

// getDefaultGateway parses the system route table to find the default gateway on Linux
func getDefaultGateway() (string, error) {
	return "", nil
}
