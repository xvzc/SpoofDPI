//go:build linux

package netutil

import (
	"fmt"
	"os/exec"
	"strings"
)

// getDefaultGateway parses the system route table to find the default gateway on Linux
func getDefaultGateway() (string, error) {
	// Use ip route to get the default route on Linux
	cmd := exec.Command("ip", "route", "show", "default")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse output: "default via 192.168.0.1 dev enp12s0 ..."
	fields := strings.Fields(string(out))
	for i, field := range fields {
		if field == "via" && i+1 < len(fields) {
			return fields[i+1], nil
		}
	}

	return "", fmt.Errorf("could not parse gateway from ip route output: %s", string(out))
}
