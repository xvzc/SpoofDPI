//go:build darwin

package tun

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/spoofdpi/internal/netutil"
	"golang.zx2c4.com/wireguard/tun"
)

const stateFile = "/tmp/spoofdpi.darwin.tun.state"

type tunStateDarwin struct {
	GatewayIP     string    `json:"gatewayIP"`
	PhysIfaceName string    `json:"physIfaceName"`
	TUNName       string    `json:"tunName"`
	CreatedAt     time.Time `json:"createdAt"`
}

func saveState(state tunStateDarwin) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(stateFile, data, 0o644)
}

func loadState() (tunStateDarwin, bool, error) {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return tunStateDarwin{}, false, nil
		}
		return tunStateDarwin{}, false, err
	}
	var state tunStateDarwin
	if err := json.Unmarshal(data, &state); err != nil {
		return tunStateDarwin{}, false, err
	}
	return state, true, nil
}

func deleteState() error {
	if err := os.Remove(stateFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// tunSystemNetworkDarwin implements TUNSystemNetwork for Darwin
type tunSystemNetworkDarwin struct {
	logger       zerolog.Logger
	tunDevice    tun.Device
	defaultRoute *netutil.Route
}

// NewTUNSystemNetwork creates a new TUNSystemNetwork for TUN mode on Darwin
// fibID is ignored on Darwin (FreeBSD-specific)
func NewTUNSystemNetwork(
	logger zerolog.Logger,
	defaultRoute *netutil.Route,
	fibID int,
) TUNSystemNetwork {
	dev, err := createTunDevice()
	if err != nil {
		return nil
	}

	return &tunSystemNetworkDarwin{
		logger:       logger,
		tunDevice:    dev,
		defaultRoute: defaultRoute,
	}
}

func (n *tunSystemNetworkDarwin) TunDevice() tun.Device {
	return n.tunDevice
}

func (n *tunSystemNetworkDarwin) DefaultRoute() *netutil.Route {
	return n.defaultRoute
}

func (n *tunSystemNetworkDarwin) SetNetworkConfig() error {
	if state, exists, err := loadState(); err == nil && exists {
		n.logger.Info().Str("iface", state.TUNName).Msg("cleaning up stale state")
		if err := n.cleanupNetworkConfig(state); err != nil {
			return fmt.Errorf("failed to cleanup stale state: %w", err)
		}
		if err := deleteState(); err != nil {
			return fmt.Errorf("failed to delete stale state: %w", err)
		}
	}

	cidr, err := netutil.FindSafeCIDR()
	if err != nil {
		return fmt.Errorf("failed to find safe subnet: %w", err)
	}

	newState := tunStateDarwin{}

	newState.GatewayIP = n.defaultRoute.Gateway.String()
	newState.PhysIfaceName = n.defaultRoute.Iface.Name
	newState.TUNName, _ = n.tunDevice.Name()

	local, _ := netutil.AddrInCIDR(cidr, 1)
	remote, _ := netutil.AddrInCIDR(cidr, 2)

	if err := setupInterface(
		newState.TUNName,
		local,
		remote,
	); err != nil {
		return fmt.Errorf("failed to set interface address: %w", err)
	}

	cmd := exec.Command(
		"route",
		"add",
		"-ifscope",
		newState.PhysIfaceName,
		"default",
		newState.GatewayIP,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		if !strings.Contains(string(out), "File exists") {
			return fmt.Errorf("failed to add scoped default route: %s: %w", string(out), err)
		}
	}

	cmd = exec.Command(
		"route",
		"-n",
		"add",
		"-host",
		newState.GatewayIP,
		"-interface",
		newState.PhysIfaceName,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		if !strings.Contains(string(out), "File exists") {
			_ = out
		}
	}

	subnets := []string{"0.0.0.0/0"}
	if err := addRoute(newState.TUNName, subnets); err != nil {
		return fmt.Errorf("failed to set default route: %w", err)
	}

	newState.CreatedAt = time.Now()
	return saveState(newState)
}

func (n *tunSystemNetworkDarwin) UnsetNetworkConfig() error {
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

func (n *tunSystemNetworkDarwin) cleanupNetworkConfig(state tunStateDarwin) error {
	subnets := []string{"0.0.0.0/0"}
	if err := deleteRoute(state.TUNName, subnets); err != nil {
		return fmt.Errorf("failed to unset default route: %w", err)
	}

	cmd := exec.Command("route", "delete", "-ifscope", state.PhysIfaceName, "default")
	if out, err := cmd.CombinedOutput(); err != nil {
		n.logger.Debug().
			Err(err).
			Str("output", string(out)).
			Msg("route delete -ifscope (ignored)")
	}

	cmd = exec.Command(
		"route",
		"-n",
		"delete",
		"-host",
		state.GatewayIP,
		"-interface",
		state.PhysIfaceName,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		n.logger.Debug().
			Err(err).
			Str("output", string(out)).
			Msg("route -n delete -host (ignored)")
	}

	return nil
}

func (n *tunSystemNetworkDarwin) BindDialer(
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

	// Set IP_BOUND_IF to bind socket to interface
	dialer.Control = func(network, address string, c syscall.RawConn) error {
		var sockErr error
		err := c.Control(func(fd uintptr) {
			sockErr = syscall.SetsockoptInt(
				int(fd),
				syscall.IPPROTO_IP,
				syscall.IP_BOUND_IF,
				iface.Index,
			)
		})
		if err != nil {
			return fmt.Errorf("failed to control socket: %w", err)
		}
		if sockErr != nil {
			return fmt.Errorf("failed to set IP_BOUND_IF: %w", sockErr)
		}
		return nil
	}

	return nil
}

// addRoute configures network routes for specified subnets
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

// deleteRoute removes previously configured network routes
func deleteRoute(iface string, subnets []string) error {
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

// setupInterface configures the TUN interface with local and remote endpoints
func setupInterface(iface string, local string, remote string) error {
	cmd := exec.Command("ifconfig", iface, local, remote, "up")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set interface address: %s: %w", string(out), err)
	}
	return nil
}

func createTunDevice() (tun.Device, error) {
	return tun.CreateTUN("utun", 1500)
}
