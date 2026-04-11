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
	cmd := exec.Command("route", "add", "-host", gateway, "-interface", iface)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "File exists") {
			return nil
		}
		return fmt.Errorf("failed to add host route to gateway: %s: %w", string(out), err)
	}
	return nil
}

func UnsetGatewayRoute(gateway, iface string) error {
	cmd := exec.Command("route", "delete", "-host", gateway, "-interface", iface)
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = out
	}
	return nil
}
