//go:build freebsd

package tun

import (
	"fmt"
	"os/exec"
	"strings"
)

func SetRoute(iface string, subnets []string) error {
	for _, subnet := range subnets {
		targets := []string{subnet}

		if subnet == "0.0.0.0/0" {
			targets = []string{"0.0.0.0/1", "128.0.0.0/1"}
		}

		for _, target := range targets {
			cmd := exec.Command("route", "-n", "add", "-net", target, "-interface", iface)
			out, err := cmd.CombinedOutput()
			if err != nil {
				if strings.Contains(string(out), "must be root") {
					return fmt.Errorf(
						"permission denied: must run as root to modify routing table (sudo required)",
					)
				}
				if strings.Contains(string(out), "File exists") {
					continue
				}
				return fmt.Errorf(
					"failed to add route for %s on %s: %s: %w",
					target,
					iface,
					string(out),
					err,
				)
			}
		}
	}
	return nil
}

func UnsetRoute(iface string, subnets []string) error {
	for _, subnet := range subnets {
		targets := []string{subnet}

		if subnet == "0.0.0.0/0" {
			targets = []string{"0.0.0.0/1", "128.0.0.0/1"}
		}

		for _, target := range targets {
			cmd := exec.Command("route", "-n", "delete", "-net", target, "-interface", iface)
			if out, err := cmd.CombinedOutput(); err != nil {
				_ = out
			}
		}
	}
	return nil
}

func SetInterfaceAddress(iface string, local string, remote string) error {
	cmd := exec.Command("ifconfig", iface, local, remote, "up")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set interface address: %s: %w", string(out), err)
	}
	return nil
}

func SetGatewayRoute(gateway, iface string) error {
	// Add a scoped default route for the physical interface
	// When a socket is bound to this interface via IP_BOUND_IF,
	// this scoped route will be used instead of the TUN routes (0/1, 128/1)
	cmd := exec.Command("route", "add", "-ifscope", iface, "default", gateway)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Ignore "File exists" error - route already exists
		if !strings.Contains(string(out), "File exists") {
			return fmt.Errorf("failed to add scoped default route: %s: %w", string(out), err)
		}
	}

	// Also add a host route to the gateway via the physical interface
	// This ensures packets to the gateway itself go through the right interface
	cmd = exec.Command("route", "-n", "add", "-host", gateway, "-interface", iface)
	out, err = cmd.CombinedOutput()
	if err != nil {
		// Ignore "File exists" error
		if !strings.Contains(string(out), "File exists") {
			// This is optional, don't fail if it doesn't work
			_ = out
		}
	}

	return nil
}

func UnsetGatewayRoute(gateway, iface string) error {
	// Remove the scoped default route for the physical interface
	// This undoes the SetGatewayRoute ifscope route
	cmd := exec.Command("route", "delete", "-ifscope", iface, "default")
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = out
	}

	// Remove the direct host route to the gateway
	cmd = exec.Command("route", "-n", "delete", "-host", gateway, "-interface", iface)
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = out
	}

	return nil
}
