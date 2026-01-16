//go:build darwin || freebsd

package netutil

import (
	"fmt"
	"os/exec"
	"regexp"
)

// getDefaultGateway parses the system route table to find the default gateway on macOS/BSD
func getDefaultGateway() (string, error) {
	// Use route to get the default route on macOS
	cmd := exec.Command("route", "-n", "get", "default")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse output to find gateway line
	re := regexp.MustCompile(`gateway:\s+(\d+\.\d+\.\d+\.\d+)`)
	matches := re.FindSubmatch(out)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not parse gateway from route output")
	}

	return string(matches[1]), nil
}
