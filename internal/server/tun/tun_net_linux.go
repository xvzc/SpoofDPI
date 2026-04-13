//go:build linux

package tun

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/spoofdpi/internal/executil"
	"github.com/xvzc/spoofdpi/internal/netutil"
	"golang.zx2c4.com/wireguard/tun"
)

const stateFile = "/tmp/spoofdpi.linux.tun.state"

type tunStateLinux struct {
	RouteTableID     int       `json:"routeTableID"`
	GatewayIP        string    `json:"gatewayIP"`
	TUNName          string    `json:"tunName"`
	PhysIfaceName    string    `json:"physIfaceName"`
	PhysIfaceIP      string    `json:"ifaceIP"`
	TunLocalIP       string    `json:"tunLocalIP"`
	TunRemoteIP      string    `json:"tunRemoteIP"`
	RouteTargetCIDRS []string  `json:"routeTargetCIDRs"`
	CreatedAt        time.Time `json:"createdAt"`
}

func saveState(state *tunStateLinux) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(stateFile, data, 0o644)
}

func loadState() (*tunStateLinux, bool, error) {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	var state tunStateLinux
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, false, err
	}
	return &state, true, nil
}

func deleteState() error {
	if err := os.Remove(stateFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (n *tunSystemNetworkLinux) cleanupNetworkConfig(state *tunStateLinux) error {
	for _, target := range state.RouteTargetCIDRS {
		if out, err := executil.Commandf("ip route del %s dev %s",
			target,
			state.TUNName,
		); err != nil {
			n.logger.Debug().Err(err).Str("output", out).Msg("ip route del (ignored)")
		}
	}

	if out, err := executil.Commandf("ip rule del from %s lookup %d",
		state.PhysIfaceIP,
		state.RouteTableID,
	); err != nil {
		n.logger.Debug().Err(err).Str("output", out).Msg("ip rule del (ignored)")
	}

	if out, err := executil.Commandf("ip route del default table %d",
		state.RouteTableID,
	); err != nil {
		n.logger.Debug().Err(err).Str("output", out).Msg("ip route del table (ignored)")
	}

	if out, err := executil.Commandf("ip route del %s dev %s",
		state.GatewayIP,
		state.PhysIfaceName,
	); err != nil {
		n.logger.Debug().Err(err).Str("output", out).Msg("ip route del gateway (ignored)")
	}

	if out, err := executil.Commandf("ip link delete %s",
		state.TUNName,
	); err != nil {
		n.logger.Debug().Err(err).Str("output", out).Msg("ip link delete <tun> (ignored)")
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
) (TUNSystemNetwork, error) {
	dev, err := createTunDevice()
	if err != nil {
		return nil, err
	}

	return &tunSystemNetworkLinux{
		logger:       logger,
		tunDevice:    dev,
		defaultRoute: defaultRoute,
	}, nil
}

func (n *tunSystemNetworkLinux) TunDevice() tun.Device {
	return n.tunDevice
}

func (n *tunSystemNetworkLinux) DefaultRoute() *netutil.Route {
	return n.defaultRoute
}

func (n *tunSystemNetworkLinux) createState() (*tunStateLinux, error) {
	var err error

	tunName, err := n.tunDevice.Name()
	if err != nil {
		return nil, fmt.Errorf("failed to get tunName: %w", err)
	}
	routeTableID, err := getOrAllocateTableID()
	if err != nil {
		return nil, fmt.Errorf("failed to allocate routing table ID: %w", err)
	}

	gatewayIP := n.defaultRoute.Gateway.String()
	physIfaceName := n.defaultRoute.Iface.Name

	physIfaceIP, err := getInterfaceIP(physIfaceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get IP address for interface %s: %w",
			physIfaceName,
			err,
		)
	}

	cidr, err := netutil.FindSafeCIDR()
	if err != nil {
		return nil, fmt.Errorf("failed to find safe subnet: %w", err)
	}

	tunLocalIP, err := netutil.AddrInCIDR(cidr, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to get %dth ip in %s: %w", 1, cidr, err)
	}
	tunRemoteIP, err := netutil.AddrInCIDR(cidr, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to get %dth ip in %s: %w", 2, cidr, err)
	}

	_, tunCIDR, _ := net.ParseCIDR(tunLocalIP + "/30")
	routeTargetCIDRS := []string{tunCIDR.String(), "0.0.0.0/1", "128.0.0.0/1"}

	state := &tunStateLinux{ //exhaustruct:enforce
		RouteTableID:     routeTableID,
		GatewayIP:        gatewayIP,
		TUNName:          tunName,
		PhysIfaceName:    physIfaceName,
		PhysIfaceIP:      physIfaceIP,
		TunLocalIP:       tunLocalIP,
		TunRemoteIP:      tunRemoteIP,
		RouteTargetCIDRS: routeTargetCIDRS,
		CreatedAt:        time.Now(),
	}
	return state, nil
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

	newState, err := n.createState()
	if err != nil {
		return err
	}

	if err := saveState(newState); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	if out, err := executil.Commandf("ip addr add %s peer %s dev %s",
		newState.TunLocalIP, newState.TunRemoteIP, newState.TUNName,
	); err != nil {
		return fmt.Errorf("failed to set interface address: %s: %w", out, err)
	}

	if out, err := executil.Commandf("ip link set dev %s up",
		newState.TUNName,
	); err != nil {
		return fmt.Errorf("failed to bring interface up: %s: %w", out, err)
	}

	if out, err := executil.Commandf("ip route add %s dev %s",
		newState.GatewayIP, newState.PhysIfaceName,
	); err != nil {
		return fmt.Errorf("failed to add gateway route: %s: %w", out, err)
	}

	if _, err := executil.Commandf("ip route add default via %s dev %s table %d",
		newState.GatewayIP, newState.PhysIfaceName, newState.RouteTableID,
	); err != nil {
		return err
	}

	if _, err := executil.Commandf("ip rule add from %s lookup %d",
		newState.PhysIfaceIP, newState.RouteTableID,
	); err != nil {
		return fmt.Errorf("failed to add policy rule: %w", err)
	}

	for _, target := range newState.RouteTargetCIDRS {
		if out, err := executil.Commandf("ip route add %s dev %s",
			target, newState.TUNName,
		); err != nil {
			return fmt.Errorf("failed to add route for %s: %w: %s", target, err, out)
		}
	}

	return nil
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
		out, err := executil.Commandf("ip route show table %d", id)
		if err != nil {
			return id, nil
		}

		if len(out) > 0 {
			continue
		}

		ruleTableOut, _ := executil.Commandf("ip rule show table %d", id)
		ruleLookupOut, _ := executil.Commandf("ip rule show lookup %d", id)

		if len(ruleTableOut) > 0 || len(ruleLookupOut) > 0 {
			continue
		}

		return id, nil
	}
	return 0, fmt.Errorf("no available routing table ID in range 200-250")
}

// getInterfaceIP returns the first IPv4 address of the given interface
func getInterfaceIP(ifaceName string) (string, error) {
	out, err := executil.Commandf("ip -4 addr show %s", ifaceName)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "inet ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				ip := strings.Split(parts[1], "/")[0]
				return ip, nil
			}
		}
	}
	return "", fmt.Errorf("IP not found for interface %s", ifaceName)
}

func createTunDevice() (tun.Device, error) {
	return tun.CreateTUN("tun%d", 1500)
}
