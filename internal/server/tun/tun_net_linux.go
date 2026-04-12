//go:build linux

package tun

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/spoofdpi/internal/netutil"
	"golang.zx2c4.com/wireguard/tun"
)

const stateFile = "/tmp/spoofdpi.linux.tun.state"

type tunStateLinux struct {
	RouteTableID  int       `json:"routeTableID"`
	GatewayIP     string    `json:"gatewayIP"`
	TUNName       string    `json:"tunName"`
	PhysIfaceName string    `json:"physIfaceName"`
	PhysIfaceIP   string    `json:"ifaceIP"`
	CreatedAt     time.Time `json:"createdAt"`
}

func saveState(state tunStateLinux) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(stateFile, data, 0o644)
}

func loadState() (tunStateLinux, bool, error) {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return tunStateLinux{}, false, nil
		}
		return tunStateLinux{}, false, err
	}
	var state tunStateLinux
	if err := json.Unmarshal(data, &state); err != nil {
		return tunStateLinux{}, false, err
	}
	return state, true, nil
}

func deleteState() error {
	if err := os.Remove(stateFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (n *tunSystemNetworkLinux) cleanupNetworkConfig(state tunStateLinux) error {
	subnets := []string{"0.0.0.0/0"}
	if err := deleteRoute(state.TUNName, subnets); err != nil {
		return fmt.Errorf("failed to unset default route: %w", err)
	}

	cmd := exec.Command(
		"ip",
		"rule",
		"del",
		"from",
		state.PhysIfaceIP,
		"lookup",
		strconv.Itoa(state.RouteTableID),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		n.logger.Debug().Err(err).Str("output", string(out)).Msg("ip rule del (ignored)")
	}

	cmd = exec.Command(
		"ip",
		"route",
		"del",
		"default",
		"table",
		strconv.Itoa(state.RouteTableID),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		n.logger.Debug().
			Err(err).
			Str("output", string(out)).
			Msg("ip route del table (ignored)")
	}

	cmd = exec.Command("ip", "route", "del", state.GatewayIP, "dev", state.PhysIfaceName)
	if out, err := cmd.CombinedOutput(); err != nil {
		n.logger.Debug().
			Err(err).
			Str("output", string(out)).
			Msg("ip route del gateway (ignored)")
	}

	return nil
}

var (
	allocatedTableID   int
	allocatedTableOnce sync.Once
)

// tunSystemNetworkLinux implements TUNSystemNetwork for Linux
type tunSystemNetworkLinux struct {
	logger       zerolog.Logger
	tunDevice    tun.Device
	defaultRoute *netutil.Route
}

// NewTUNSystemNetwork creates a new TUNSystemNetwork for TUN mode on Linux
// fibID is ignored on Linux (FreeBSD-specific)
func NewTUNSystemNetwork(
	logger zerolog.Logger,
	defaultRoute *netutil.Route,
	fibID int,
) TUNSystemNetwork {
	dev, err := createTunDevice()
	if err != nil {
		return nil
	}

	return &tunSystemNetworkLinux{
		logger:       logger,
		tunDevice:    dev,
		defaultRoute: defaultRoute,
	}
}

func (n *tunSystemNetworkLinux) TunDevice() tun.Device {
	return n.tunDevice
}

func (n *tunSystemNetworkLinux) DefaultRoute() *netutil.Route {
	return n.defaultRoute
}

func (n *tunSystemNetworkLinux) SetNetworkConfig() error {
	if state, exists, err := loadState(); err == nil && exists {
		if err := n.cleanupNetworkConfig(state); err != nil {
			return fmt.Errorf("failed to cleanup stale state: %w", err)
		}
		if err := deleteState(); err != nil {
			return fmt.Errorf("failed to delete stale state: %w", err)
		}
	}

	var err error

	newState := tunStateLinux{}
	newState.TUNName, _ = n.tunDevice.Name()
	newState.RouteTableID, err = getOrAllocateTableID()
	if err != nil {
		return fmt.Errorf("failed to allocate routing table ID: %w", err)
	}

	newState.GatewayIP = n.defaultRoute.Gateway.String()
	newState.PhysIfaceName = n.defaultRoute.Iface.Name

	newState.PhysIfaceIP = getInterfaceIP(newState.PhysIfaceName)
	if newState.PhysIfaceIP == "" {
		return fmt.Errorf(
			"failed to get IP address for interface %s",
			newState.PhysIfaceName,
		)
	}

	cidr, err := netutil.FindSafeCIDR()
	if err != nil {
		return fmt.Errorf("failed to find safe subnet: %w", err)
	}
	local, _ := netutil.AddrInCIDR(cidr, 1)
	remote, _ := netutil.AddrInCIDR(cidr, 2)

	if err := setupInterface(newState.TUNName, local, remote); err != nil {
		return fmt.Errorf("failed to set interface address: %w", err)
	}

	cmd := exec.Command(
		"ip",
		"route",
		"add",
		newState.GatewayIP,
		"dev",
		newState.PhysIfaceName,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		if !strings.Contains(string(out), "File exists") {
			return fmt.Errorf("failed to add gateway route: %s: %w", string(out), err)
		}
	}

	cmd = exec.Command(
		"ip",
		"route",
		"add",
		"default",
		"via",
		newState.GatewayIP,
		"dev",
		newState.PhysIfaceName,
		"table",
		strconv.Itoa(newState.RouteTableID),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		if !strings.Contains(string(out), "File exists") {
			_ = out
		}
	}

	cmd = exec.Command(
		"ip",
		"rule",
		"add",
		"from",
		newState.PhysIfaceIP,
		"lookup",
		strconv.Itoa(newState.RouteTableID),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		if !strings.Contains(string(out), "File exists") {
			return fmt.Errorf("failed to add policy rule: %s: %w", string(out), err)
		}
	}

	localIP := net.ParseIP(local)
	networkAddr := net.IPv4(
		localIP[12],
		localIP[13],
		localIP[14],
		localIP[15]&0xFC,
	)

	if err := addRoute(
		newState.TUNName,
		[]string{networkAddr.String() + "/30"},
	); err != nil {
		return fmt.Errorf("failed to set local route: %w", err)
	}

	if err := addRoute(newState.TUNName, []string{"0.0.0.0/0"}); err != nil {
		return fmt.Errorf("failed to set default route: %w", err)
	}

	newState.CreatedAt = time.Now()
	return saveState(newState)
}

func (n *tunSystemNetworkLinux) UnsetNetworkConfig() error {
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

func (n *tunSystemNetworkLinux) BindDialer(
	dialer *net.Dialer,
	network string,
	targetIP net.IP,
) error {
	if n.defaultRoute == nil || n.defaultRoute.Iface.Name == "" {
		return nil
	}

	iface := n.defaultRoute.Iface

	// Find the interface's IP address to use as source
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

	// Set SO_BINDTODEVICE to bind socket to interface
	dialer.Control = func(network, address string, c syscall.RawConn) error {
		var sockErr error
		err := c.Control(func(fd uintptr) {
			sockErr = syscall.SetsockoptString(
				int(fd),
				syscall.SOL_SOCKET,
				syscall.SO_BINDTODEVICE,
				n.defaultRoute.Iface.Name,
			)
		})
		if err != nil {
			return fmt.Errorf("failed to control socket: %w", err)
		}
		if sockErr != nil {
			return fmt.Errorf("failed to set SO_BINDTODEVICE: %w", sockErr)
		}
		return nil
	}

	return nil
}

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

// findAvailableTableID finds an unused routing table ID in the range 200-250.
func findAvailableTableID() (int, error) {
	for id := 200; id <= 250; id++ {
		cmd := exec.Command("ip", "route", "show", "table", strconv.Itoa(id))
		out, err := cmd.CombinedOutput()
		if err != nil {
			return id, nil
		}

		if len(out) > 0 {
			continue
		}

		ruleTableCmd := exec.Command("ip", "rule", "show", "table", strconv.Itoa(id))
		ruleTableOut, _ := ruleTableCmd.CombinedOutput()

		ruleLookupCmd := exec.Command("ip", "rule", "show", "lookup", strconv.Itoa(id))
		ruleLookupOut, _ := ruleLookupCmd.CombinedOutput()

		if len(ruleTableOut) > 0 || len(ruleLookupOut) > 0 {
			continue
		}

		return id, nil
	}
	return 0, fmt.Errorf("no available routing table ID in range 200-250")
}

// addRoute configures network routes for specified subnets using ip route
func addRoute(iface string, subnets []string) error {
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

// deleteRoute removes previously configured network routes using ip route
func deleteRoute(iface string, subnets []string) error {
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

// setupInterface configures the TUN interface with local and remote endpoints using ip addr
func setupInterface(iface string, local string, remote string) error {
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

func createTunDevice() (tun.Device, error) {
	return tun.CreateTUN("tun", 1500)
}
