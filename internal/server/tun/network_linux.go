//go:build linux

package tun

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

var (
	allocatedTableID   int
	allocatedTableOnce sync.Once
)

// getOrAllocateTableID returns a routing table ID, allocating one if needed.
// The ID is cached after first allocation.
func getOrAllocateTableID() (int, error) {
	var initErr error
	allocatedTableOnce.Do(func() {
		allocatedTableID, initErr = findAvailableTableID()
	})
	if allocatedTableID == 0 {
		return 0, initErr
	}
	return allocatedTableID, nil
}

// findAvailableTableID finds an unused routing table ID in the range 100-252.
func findAvailableTableID() (int, error) {
	usedTables := make(map[int]bool)

	// Parse "ip rule show" output to find used table IDs
	cmd := exec.Command("ip", "rule", "show")
	if out, err := cmd.Output(); err == nil {
		re := regexp.MustCompile(`lookup\s+(\d+)`)
		matches := re.FindAllStringSubmatch(string(out), -1)
		for _, match := range matches {
			if len(match) >= 2 {
				if id, err := strconv.Atoi(match[1]); err == nil {
					usedTables[id] = true
				}
			}
		}
	}

	// Also check /etc/iproute2/rt_tables for reserved tables
	rtTablesCmd := exec.Command("cat", "/etc/iproute2/rt_tables")
	if rtOut, err := rtTablesCmd.Output(); err == nil {
		lines := strings.Split(string(rtOut), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 1 {
				if id, err := strconv.Atoi(fields[0]); err == nil {
					usedTables[id] = true
				}
			}
		}
	}

	// Find first available ID in range 100-252 (253-255 are reserved)
	for id := 100; id <= 252; id++ {
		if !usedTables[id] {
			return id, nil
		}
	}

	return 0, fmt.Errorf("no available routing table ID in range 100-252")
}

// SetRoute configures network routes for specified subnets using ip route
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
			cmd := exec.Command("ip", "route", "add", target, "dev", iface)
			out, err := cmd.CombinedOutput()
			if err != nil {
				// Check if it's a permission error
				if strings.Contains(string(out), "Operation not permitted") ||
					strings.Contains(string(out), "RTNETLINK answers: Operation not permitted") {
					return fmt.Errorf(
						"permission denied: must run as root to modify routing table (sudo required)",
					)
				}
				// Ignore "File exists" error - route already exists
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

// UnsetRoute removes previously configured network routes using ip route
func UnsetRoute(iface string, subnets []string) error {
	for _, subnet := range subnets {
		targets := []string{subnet}

		if subnet == "0.0.0.0/0" {
			targets = []string{"0.0.0.0/1", "128.0.0.0/1"}
		}

		for _, target := range targets {
			/* Delete specific routes to revert traffic flow to the original gateway. */
			cmd := exec.Command("ip", "route", "del", target, "dev", iface)
			if out, err := cmd.CombinedOutput(); err != nil {
				_ = out
			}
		}
	}
	return nil
}

// SetInterfaceAddress configures the TUN interface with local and remote endpoints using ip addr
func SetInterfaceAddress(iface string, local string, remote string) error {
	// Add the IP address to the interface
	cmd := exec.Command("ip", "addr", "add", local, "peer", remote, "dev", iface)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Ignore if address already exists
		if !strings.Contains(string(out), "File exists") {
			return fmt.Errorf("failed to set interface address: %s: %w", string(out), err)
		}
	}

	// Bring the interface up
	cmd = exec.Command("ip", "link", "set", "dev", iface, "up")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bring interface up: %s: %w", string(out), err)
	}

	return nil
}

// UnsetGatewayRoute removes the gateway route and policy routing rules
func UnsetGatewayRoute(gateway, iface string) error {
	tableID, err := getOrAllocateTableID()
	if err != nil {
		return err
	}
	tableIDStr := strconv.Itoa(tableID)

	// Get the interface's IP address for policy routing cleanup
	ifaceIP := getInterfaceIP(iface)
	if ifaceIP != "" {
		// Remove the policy rule
		cmd := exec.Command("ip", "rule", "del", "from", ifaceIP, "lookup", tableIDStr)
		if out, err := cmd.CombinedOutput(); err != nil {
			_ = out
		}
	}

	// Remove the route from the allocated table
	cmd := exec.Command("ip", "route", "del", "default", "table", tableIDStr)
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = out
	}

	// Remove the direct route to the gateway
	cmd = exec.Command("ip", "route", "del", gateway, "dev", iface)
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = out
	}

	return nil
}

// SetGatewayRoute configures policy routing so that packets from the physical interface
// are routed via the gateway, while other packets go through TUN
func SetGatewayRoute(gateway, iface string) error {
	tableID, err := getOrAllocateTableID()
	if err != nil {
		return fmt.Errorf("failed to allocate routing table ID: %w", err)
	}
	tableIDStr := strconv.Itoa(tableID)

	// First, add a direct route to the gateway via the interface
	cmd := exec.Command("ip", "route", "add", gateway, "dev", iface)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Ignore "File exists" error - route already exists
		if !strings.Contains(string(out), "File exists") {
			return fmt.Errorf("failed to add gateway route: %s: %w", string(out), err)
		}
	}

	// Add default route to the allocated table via the gateway
	cmd = exec.Command("ip", "route", "add", "default", "via", gateway, "dev", iface, "table", tableIDStr)
	out, err = cmd.CombinedOutput()
	if err != nil {
		if !strings.Contains(string(out), "File exists") {
			// Optional, don't fail
			_ = out
		}
	}

	// Get the interface's IP address for policy routing
	ifaceIP := getInterfaceIP(iface)
	if ifaceIP == "" {
		return fmt.Errorf("failed to get IP address for interface %s", iface)
	}

	// Add policy rule: packets from this IP use the allocated table
	cmd = exec.Command("ip", "rule", "add", "from", ifaceIP, "lookup", tableIDStr)
	out, err = cmd.CombinedOutput()
	if err != nil {
		if !strings.Contains(string(out), "File exists") {
			return fmt.Errorf("failed to add policy rule: %s: %w", string(out), err)
		}
	}

	return nil
}

// getInterfaceIP returns the first IPv4 address of the given interface
func getInterfaceIP(ifaceName string) string {
	cmd := exec.Command("ip", "-4", "addr", "show", ifaceName)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Parse output to find inet line
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "inet ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				ip := strings.Split(parts[1], "/")[0]
				return ip
			}
		}
	}
	return ""
}
