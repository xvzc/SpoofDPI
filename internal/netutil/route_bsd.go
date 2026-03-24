//go:build darwin || freebsd

package netutil

import (
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"syscall"

	"golang.org/x/sys/unix"
)

// bindToInterface sets the dialer's Control function to bind the socket
// to a specific network interface using IP_BOUND_IF on BSD systems.
func bindToInterface(
	network string,
	dialer *net.Dialer,
	iface *net.Interface,
	targetIP net.IP,
) error {
	if iface == nil {
		return nil
	}

	ifaceIndex := iface.Index
	dialer.Control = func(network, address string, c syscall.RawConn) error {
		var setsockoptErr error
		err := c.Control(func(fd uintptr) {
			setsockoptErr = unix.SetsockoptInt(
				int(fd),
				unix.IPPROTO_IP,
				unix.IP_BOUND_IF,
				ifaceIndex,
			)
		})
		if err != nil {
			return err
		}
		return setsockoptErr
	}
	return nil
}

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
