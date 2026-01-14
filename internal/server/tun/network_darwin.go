package tun

import (
	"fmt"
	"os/exec"
	"strings"
)

// SetRoute configures network routes for specified subnets
func SetRoute(iface string, subnets []string) error {
	for _, subnet := range subnets {
		targets := []string{subnet}

		/* Expand default route into two /1 subnets to override the default gateway
		   without removing the existing 0.0.0.0/0 entry.
		*/
		if subnet == "0.0.0.0/0" {
			targets = []string{"0.0.0.0/1", "128.0.0.0/1"}
		}

		for _, target := range targets {
			cmd := exec.Command("route", "-n", "add", "-net", target, "-interface", iface)
			out, err := cmd.CombinedOutput()
			if err != nil {
				// Check if it's a permission error
				if strings.Contains(string(out), "must be root") {
					return fmt.Errorf(
						"permission denied: must run as root to modify routing table (sudo required)",
					)
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

// UnsetRoute removes previously configured network routes
func UnsetRoute(iface string, subnets []string) error {
	for _, subnet := range subnets {
		targets := []string{subnet}

		if subnet == "0.0.0.0/0" {
			targets = []string{"0.0.0.0/1", "128.0.0.0/1"}
		}

		for _, target := range targets {
			/* Delete specific routes to revert traffic flow to the original gateway. */
			cmd := exec.Command("route", "-n", "delete", "-net", target, "-interface", iface)
			if out, err := cmd.CombinedOutput(); err != nil {
				_ = out
			}
		}
	}
	return nil
}

// SetInterfaceAddress configures the TUN interface with local and remote endpoints
func SetInterfaceAddress(iface string, local string, remote string) error {
	cmd := exec.Command("ifconfig", iface, local, remote, "up")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set interface address: %s: %w", string(out), err)
	}
	return nil
}

// UnsetGatewayRoute removes the gateway route
func UnsetGatewayRoute(gateway, iface string) error {
	// Remove the direct route to the gateway
	cmd := exec.Command("route", "-n", "delete", "-host", gateway, "-interface", iface)
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = out
	}

	// Also try to remove the 0.0.0.0/2 route if it exists (cleanup from previous versions)
	cmd = exec.Command("route", "-n", "delete", "-net", "0.0.0.0/32", gateway)
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = out
	}

	return nil
}

// SetGatewayRoute adds a host route to the gateway via the specified interface
// This ensures traffic destined for the gateway goes through the physical interface
func SetGatewayRoute(gateway, iface string) error {
	// First, get the gateway's subnet to add a direct route
	cmd := exec.Command("route", "-n", "add", "-host", gateway, "-interface", iface)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Ignore "File exists" error - route already exists
		if !strings.Contains(string(out), "File exists") {
			return fmt.Errorf("failed to add gateway route: %s: %w", string(out), err)
		}
	}

	// Also add a less specific route that uses the gateway for 0.0.0.0/2
	// This provides a path for IP_BOUND_IF sockets on en0 to reach external hosts
	// The 0/2 route is less specific than 0/1 but will be used when bound to en0
	cmd = exec.Command("route", "-n", "add", "-net", "0.0.0.0/32", gateway)
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
