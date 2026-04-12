//go:build freebsd

package tun

/*

Package tun provides FreeBSD-specific TUN networking functionality.

Route Management Logic:

 1. SetGatewayRoute: Set up gateway routing for a FIB
    - Load state file and verify current routes against actual system state
    - If no state file: FIB must be empty (error if any routes exist)
    - If state file exists: routes must exactly match recorded state
    - Clean up old routes if verified to be ours
    - Add new subnet and default routes
    - Save new state to /tmp/spoofdpi.fib

 2. UnsetGatewayRoute: Clean up gateway routing
    - Load state file to find recorded routes
    - Delete default route from recorded FIB
    - Delete subnet route from recorded FIB
    - Remove state file

State File Format (JSON):
	{
		"fib_id": 2,
		"gateway":
		"192.168.1.1",
		"iface": "tun0",
		"subnet": "192.168.1.0/24",
		"created_at": "2026-04-12T10:30:00Z"
	}

*/

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/spoofdpi/internal/netutil"
	"golang.zx2c4.com/wireguard/tun"
)

const (
	stateFilePath = "/tmp/spoofdpi.freebsd.tun.state"
)

// tunStateFreeBSD represents the saved state of FIB configuration
type tunStateFreeBSD struct {
	FIBID         int       `json:"fibID"`
	Gateway       string    `json:"gateway"`
	PhysIfaceName string    `json:"iface"`
	Subnet        string    `json:"subnet"`
	TUNName       string    `json:"tunName"`
	CreatedAt     time.Time `json:"createdAt"`
}

// tunSystemNetworkFreeBSD implements TUNSystemNetwork for FreeBSD
type tunSystemNetworkFreeBSD struct {
	logger       zerolog.Logger
	tunDevice    tun.Device
	defaultRoute *netutil.Route
	fibID        int
}

// NewTUNSystemNetwork creates a new TUNSystemNetwork for TUN mode on FreeBSD
func NewTUNSystemNetwork(
	logger zerolog.Logger,
	defaultRoute *netutil.Route,
	fibID int,
) TUNSystemNetwork {
	dev, err := createTunDevice()
	if err != nil {
		return nil
	}

	return &tunSystemNetworkFreeBSD{
		logger:       logger,
		tunDevice:    dev,
		defaultRoute: defaultRoute,
		fibID:        fibID,
	}
}

func (n *tunSystemNetworkFreeBSD) TunDevice() tun.Device {
	return n.tunDevice
}

func (n *tunSystemNetworkFreeBSD) DefaultRoute() *netutil.Route {
	return n.defaultRoute
}

func (n *tunSystemNetworkFreeBSD) SetNetworkConfig() error {
	if n.fibID <= 0 {
		return fmt.Errorf("FIB ID must be greater than 0, got %d", n.fibID)
	}

	state, exists, err := loadState()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}
	if exists {
		if err := n.cleanupNetworkConfig(state); err != nil {
			return fmt.Errorf("failed to cleanup stale state: %w", err)
		}
		if err := deleteState(); err != nil {
			return fmt.Errorf("failed to delete state: %w", err)
		}
	}

	cmd := exec.Command("setfib", strconv.Itoa(n.fibID), "route", "get", "default")
	if err := cmd.Run(); err == nil {
		return fmt.Errorf("FIB %d is already in use", n.fibID)
	}

	cidr, err := netutil.FindSafeCIDR()
	if err != nil {
		return fmt.Errorf("failed to find safe subnet: %w", err)
	}

	newState := tunStateFreeBSD{}
	newState.FIBID = n.fibID
	newState.TUNName, _ = n.tunDevice.Name()
	newState.Gateway = n.defaultRoute.Gateway.String()
	newState.PhysIfaceName = n.defaultRoute.Iface.Name
	newState.Subnet, err = getInterfaceSubnet(newState.PhysIfaceName)
	newState.CreatedAt = time.Now()
	if err != nil {
		return fmt.Errorf("failed to get interface subnet: %w", err)
	}

	local, _ := netutil.AddrInCIDR(cidr, 1)
	remote, _ := netutil.AddrInCIDR(cidr, 2)

	if err := setupInterface(newState.TUNName, local, remote); err != nil {
		return fmt.Errorf("failed to set interface address: %w", err)
	}

	if err := addFIBSubnetRoute(
		newState.FIBID,
		newState.Subnet,
		newState.PhysIfaceName,
	); err != nil {
		return fmt.Errorf("failed to add subnet route: %w", err)
	}

	if err := addFIBDefaultRoute(newState.FIBID, newState.Gateway); err != nil {
		_ = deleteFIBSubnetRoute(newState.FIBID, newState.Subnet, newState.PhysIfaceName)
		return fmt.Errorf("failed to add default route: %w", err)
	}

	subnets := []string{"0.0.0.0/0"}
	if err := addRoute(newState.TUNName, subnets); err != nil {
		return fmt.Errorf("failed to set default route: %w", err)
	}

	return saveState(newState)
}

func (n *tunSystemNetworkFreeBSD) UnsetNetworkConfig() error {
	state, exists, err := loadState()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}
	if !exists {
		return nil
	}

	if err := n.cleanupNetworkConfig(state); err != nil {
		return fmt.Errorf("failed to cleanup network config: %w", err)
	}

	return deleteState()
}

func (n *tunSystemNetworkFreeBSD) cleanupNetworkConfig(state tunStateFreeBSD) error {
	if err := deleteFIBDefaultRoute(state.FIBID); err != nil {
		n.logger.Debug().
			Err(err).
			Int("fib", state.FIBID).
			Msg("deleteFIBDefaultRoute (ignored)")
	}

	if err := deleteFIBSubnetRoute(
		state.FIBID,
		state.Subnet,
		state.PhysIfaceName,
	); err != nil {
		n.logger.Debug().
			Err(err).
			Int("fib", state.FIBID).
			Msg("deleteFIBSubnetRoute (ignored)")
	}

	if err := deleteRoute(state.TUNName, []string{"0.0.0.0/0"}); err != nil {
		n.logger.Debug().Err(err).Msg("deleteRoute tun default (ignored)")
	}

	return nil
}

func (n *tunSystemNetworkFreeBSD) BindDialer(
	dialer *net.Dialer,
	network string,
	targetIP net.IP,
) error {
	if n.fibID <= 0 || n.defaultRoute == nil || n.defaultRoute.Iface.Name == "" {
		return nil
	}

	// Find the interface's IP address to use as source
	iface := n.defaultRoute.Iface

	addrs, err := iface.Addrs()
	if err != nil {
		return fmt.Errorf("failed to get interface addresses: %w", err)
	}

	var sourceIP net.IP
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			// Match IP version: use IPv4 source for IPv4 target, IPv6 for IPv6
			if targetIP.To4() != nil && ipnet.IP.To4() != nil && !ipnet.IP.IsLoopback() {
				sourceIP = ipnet.IP
				break
			} else if targetIP.To4() == nil && ipnet.IP.To4() == nil && !ipnet.IP.IsLoopback() {
				sourceIP = ipnet.IP
				break
			}
		}
	}

	if sourceIP == nil {
		return fmt.Errorf(
			"no suitable IP address found on interface %s for target %s",
			n.defaultRoute.Iface.Name,
			targetIP,
		)
	}

	// Set the LocalAddr for the source IP
	if strings.HasPrefix(network, "tcp") {
		dialer.LocalAddr = &net.TCPAddr{IP: sourceIP}
	} else if strings.HasPrefix(network, "udp") {
		dialer.LocalAddr = &net.UDPAddr{IP: sourceIP}
	} else {
		dialer.LocalAddr = &net.IPAddr{IP: sourceIP}
	}

	// Set SO_SETFIB to use the allocated FIB
	dialer.Control = func(network, address string, c syscall.RawConn) error {
		var sockErr error
		err := c.Control(func(fd uintptr) {
			sockErr = syscall.SetsockoptInt(
				int(fd),
				syscall.SOL_SOCKET,
				syscall.SO_SETFIB,
				n.fibID,
			)
		})
		if err != nil {
			return fmt.Errorf("failed to control socket: %w", err)
		}
		if sockErr != nil {
			return fmt.Errorf("failed to set SO_SETFIB to %d: %w", n.fibID, sockErr)
		}
		return nil
	}

	return nil
}

// loadState loads the tun state from the state file
func loadState() (tunStateFreeBSD, bool, error) {
	data, err := os.ReadFile(stateFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return tunStateFreeBSD{}, false, nil
		}
		return tunStateFreeBSD{}, false, err
	}

	var state tunStateFreeBSD
	if err := json.Unmarshal(data, &state); err != nil {
		return tunStateFreeBSD{}, false, err
	}

	return state, true, nil
}

// saveState saves the tun state to the state file
func saveState(state tunStateFreeBSD) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(stateFilePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// deleteState removes the state file
func deleteState() error {
	if err := os.Remove(stateFilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove state file: %w", err)
	}
	return nil
}

// getInterfaceSubnet gets the subnet of the given interface in CIDR notation
// e.g., "10.111.111.0/24"
func getInterfaceSubnet(ifaceName string) (string, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return "", fmt.Errorf("failed to get interface %s: %w", ifaceName, err)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return "", fmt.Errorf("failed to get interface addresses: %w", err)
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
			// Get network address by masking IP with mask
			network := ipnet.IP.Mask(ipnet.Mask)
			ones, _ := ipnet.Mask.Size()
			subnet := fmt.Sprintf("%s/%d", network.String(), ones)
			return subnet, nil
		}
	}

	return "", fmt.Errorf("no IPv4 address found on interface %s", ifaceName)
}

// hasFIBDefaultRoute checks if FIB has a default route
// Returns: (hasRoute bool, isOurs bool)
func hasFIBDefaultRoute(fibID int) (bool, bool) {
	cmd := exec.Command("netstat", "-rn", "-F", strconv.Itoa(fibID))
	out, err := cmd.Output()
	if err != nil {
		return false, false
	}

	lines := strings.Split(string(out), "\n")
	hasDefault := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "default") {
			hasDefault = true
			break
		}
	}

	if !hasDefault {
		return false, false
	}

	// Check if it's our route by looking at state file
	state, exists, _ := loadState()
	if exists && state.FIBID == fibID {
		return true, true
	}

	return true, false
}

// queryFIBRoutes returns all destinations in the specified FIB
func queryFIBRoutes(fibID int) ([]string, error) {
	cmd := exec.Command("netstat", "-rn", "-F", strconv.Itoa(fibID))
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get routes: %w", err)
	}

	var routes []string
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip header lines and empty lines
		if line == "" || strings.HasPrefix(line, "Routing") ||
			strings.HasPrefix(line, "Destination") || strings.HasPrefix(line, "Internet:") {
			continue
		}

		// Parse route destination (first column)
		fields := strings.Fields(line)
		if len(fields) >= 1 {
			dest := fields[0]
			// Only include IPv4 routes (skip IPv6)
			if strings.Contains(dest, ":") {
				continue
			}
			routes = append(routes, dest)
		}
	}

	return routes, nil
}

// verifyFIBRoutes checks if FIB routes match expected state
// - state == nil: FIB must be empty
// - state != nil: FIB must contain exactly state.Gateway (default) and state.Subnet
func verifyFIBRoutes(fibID int, state *tunStateFreeBSD) error {
	routes, err := queryFIBRoutes(fibID)
	if err != nil {
		return err
	}

	if state == nil {
		// No state file - FIB must be empty
		if len(routes) > 0 {
			return fmt.Errorf("FIB %d has unexpected routes: %v", fibID, routes)
		}
		return nil
	}

	// State file exists - verify routes match exactly
	expectedRoutes := map[string]bool{
		"default":    true,
		state.Subnet: true,
	}

	// Check all expected routes exist
	for _, expected := range []string{"default", state.Subnet} {
		found := false
		for _, route := range routes {
			if route == expected {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("FIB %d missing expected route: %s", fibID, expected)
		}
	}

	// Check no unexpected routes exist
	for _, route := range routes {
		if !expectedRoutes[route] {
			return fmt.Errorf("FIB %d has unexpected route: %s", fibID, route)
		}
	}

	return nil
}

// addFIBSubnetRoute adds a subnet route to the specified FIB
func addFIBSubnetRoute(fibID int, subnet string, iface string) error {
	cmd := exec.Command(
		"route",
		"add",
		"-net",
		subnet,
		"-iface",
		iface,
		"-fib",
		strconv.Itoa(fibID),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Ignore "File exists" error
		if strings.Contains(string(out), "File exists") {
			return nil
		}
		return fmt.Errorf("failed to add subnet route: %s: %w", string(out), err)
	}
	return nil
}

// deleteFIBSubnetRoute deletes the subnet route from the specified FIB
func deleteFIBSubnetRoute(fibID int, subnet string, iface string) error {
	cmd := exec.Command(
		"route",
		"delete",
		"-net",
		subnet,
		"-iface",
		iface,
		"-fib",
		strconv.Itoa(fibID),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		// Ignore "not in table" error
		if !strings.Contains(string(out), "not in table") {
			return fmt.Errorf("failed to delete subnet route: %s: %w", string(out), err)
		}
	}
	return nil
}

// addFIBDefaultRoute adds a default route to the specified FIB
func addFIBDefaultRoute(fibID int, gateway string) error {
	cmd := exec.Command("route", "add", "default", gateway, "-fib", strconv.Itoa(fibID))
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Ignore "File exists" error
		if strings.Contains(string(out), "File exists") {
			return nil
		}
		return fmt.Errorf("failed to add default route: %s: %w", string(out), err)
	}
	return nil
}

// deleteFIBDefaultRoute deletes the default route from the specified FIB
func deleteFIBDefaultRoute(fibID int) error {
	cmd := exec.Command("route", "delete", "default", "-fib", strconv.Itoa(fibID))
	if out, err := cmd.CombinedOutput(); err != nil {
		// Ignore "not in table" error
		if !strings.Contains(string(out), "not in table") {
			return fmt.Errorf("failed to delete default route: %s: %w", string(out), err)
		}
	}
	return nil
}

func addRoute(iface string, subnets []string) error {
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

func deleteRoute(iface string, subnets []string) error {
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

func setupInterface(iface string, local string, remote string) error {
	cmd := exec.Command("ifconfig", iface, local, remote, "up")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set interface address: %s: %w", string(out), err)
	}
	return nil
}

func createTunDevice() (tun.Device, error) {
	return tun.CreateTUN("tun", 1500)
}
