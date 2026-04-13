//go:build darwin

package tun

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/spoofdpi/internal/executil"
	"github.com/xvzc/spoofdpi/internal/netutil"
	"golang.zx2c4.com/wireguard/tun"
)

const stateFile = "/tmp/spoofdpi.darwin.tun.state"

type tunStateDarwin struct {
	GatewayIP        string    `json:"gatewayIP"`
	PhysIfaceName    string    `json:"physIfaceName"`
	TUNName          string    `json:"tunName"`
	TunLocalIP       string    `json:"tunLocalIP"`
	TunRemoteIP      string    `json:"tunRemoteIP"`
	RouteTargetCIDRs []string  `json:"routeTargetCIDRs"`
	CreatedAt        time.Time `json:"createdAt"`
}

func saveState(state *tunStateDarwin) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(stateFile, data, 0o644)
}

func loadState() (*tunStateDarwin, bool, error) {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	var state tunStateDarwin
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
) (TUNSystemNetwork, error) {
	dev, err := createTunDevice()
	if err != nil {
		return nil, err
	}

	return &tunSystemNetworkDarwin{
		logger:       logger,
		tunDevice:    dev,
		defaultRoute: defaultRoute,
	}, nil
}

func (n *tunSystemNetworkDarwin) TunDevice() tun.Device {
	return n.tunDevice
}

func (n *tunSystemNetworkDarwin) DefaultRoute() *netutil.Route {
	return n.defaultRoute
}

func (n *tunSystemNetworkDarwin) createState() (*tunStateDarwin, error) {
	tunName, err := n.tunDevice.Name()
	if err != nil {
		return nil, fmt.Errorf("failed to get tunName: %w", err)
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

	state := &tunStateDarwin{ //exhaustruct:enforce
		GatewayIP:        n.defaultRoute.Gateway.String(),
		PhysIfaceName:    n.defaultRoute.Iface.Name,
		TUNName:          tunName,
		TunLocalIP:       tunLocalIP,
		TunRemoteIP:      tunRemoteIP,
		RouteTargetCIDRs: routeTargetCIDRS,
		CreatedAt:        time.Now(),
	}
	return state, nil
}

func (n *tunSystemNetworkDarwin) cleanupNetworkConfig(state *tunStateDarwin) {
	for _, target := range state.RouteTargetCIDRs {
		if out, err := executil.Commandf("route -n delete -net %s -interface %s",
			target, state.TUNName); err != nil {
			n.logger.Debug().Err(err).Str("output", out).Msg("route delete (ignored)")
		}
	}

	if out, err := executil.Commandf("route -n delete -host %s -interface %s",
		state.GatewayIP, state.PhysIfaceName); err != nil {
		n.logger.Debug().Err(err).Str("output", out).Msg("route -n delete -host (ignored)")
	}

	if out, err := executil.Commandf("route delete -ifscope %s default",
		state.PhysIfaceName); err != nil {
		n.logger.Debug().Err(err).Str("output", out).Msg("route delete -ifscope (ignored)")
	}
}

func (n *tunSystemNetworkDarwin) SetNetworkConfig() error {
	if state, exists, err := loadState(); err == nil && exists {
		n.logger.Info().Str("iface", state.TUNName).Msg("cleaning up stale state")
		n.cleanupNetworkConfig(state)
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

	if out, err := executil.Commandf("ifconfig %s %s %s up",
		newState.TUNName, newState.TunLocalIP, newState.TunRemoteIP); err != nil {
		return fmt.Errorf("failed to set interface address: %s: %w", out, err)
	}

	if out, err := executil.Commandf("route add -ifscope %s default %s",
		newState.PhysIfaceName, newState.GatewayIP); err != nil {
		return fmt.Errorf("failed to add scoped default route: %s: %w", out, err)
	}

	if out, err := executil.Commandf("route -n add -host %s -interface %s",
		newState.GatewayIP, newState.PhysIfaceName); err != nil {
		return fmt.Errorf("failed to add host route: %s: %w", out, err)
	}

	for _, target := range newState.RouteTargetCIDRs {
		if out, err := executil.Commandf("route -n add -net %s -interface %s",
			target, newState.TUNName); err != nil {
			return fmt.Errorf("failed to add route for %s: %s: %w", target, out, err)
		}
	}

	return nil
}

func (n *tunSystemNetworkDarwin) UnsetNetworkConfig() error {
	state, exists, err := loadState()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}
	if !exists {
		return nil
	}

	n.cleanupNetworkConfig(state)
	return deleteState()
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

	addrs, err := iface.Addrs()
	if err != nil {
		return fmt.Errorf("failed to get interface addresses: %w", err)
	}

	var sourceIP net.IP
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
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

	if strings.HasPrefix(network, "tcp") {
		dialer.LocalAddr = &net.TCPAddr{IP: sourceIP}
	} else if strings.HasPrefix(network, "udp") {
		dialer.LocalAddr = &net.UDPAddr{IP: sourceIP}
	} else {
		dialer.LocalAddr = &net.IPAddr{IP: sourceIP}
	}

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

func createTunDevice() (tun.Device, error) {
	return tun.CreateTUN("utun", 1500)
}

func configurationJobs(state *tunStateDarwin) []configurationJob {
	return []configurationJob{
		{
			Set: func() error {
				if out, err := executil.Commandf("ifconfig %s %s %s up",
					state.TUNName, state.TunLocalIP, state.TunRemoteIP); err != nil {
					return fmt.Errorf("failed to set interface address: %s: %w", out, err)
				}

				return nil
			},
			Unset: func() error {
				if out, err := executil.Commandf("ifconfig %s destroy",
					state.TUNName,
				); err != nil {
					return fmt.Errorf("failed to set interface address: %s: %w", out, err)
				}

				return nil
			},
		},
		{
			Set: func() error {
				if out, err := executil.Commandf("route add -ifscope %s default %s",
					state.PhysIfaceName, state.GatewayIP); err != nil {
					return fmt.Errorf("failed to add scoped default route: %s: %w", out, err)
				}

				return nil
			},
			Unset: func() error {
				if out, err := executil.Commandf("route delete -ifscope %s default %s",
					state.PhysIfaceName, state.GatewayIP,
				); err != nil {
					n.logger.Debug().
						Err(err).
						Str("output", out).
						Msg("route delete -ifscope (ignored)")
				}

				return nil
			},
		},
		{
			Set: func() error {
				if out, err := executil.Commandf("route -n add -host %s -interface %s",
					state.GatewayIP, state.PhysIfaceName,
				); err != nil {
					return fmt.Errorf("failed to add host route: %s: %w", out, err)
				}

				return nil
			},
			Unset: func() error {
				if out, err := executil.Commandf("route -n delete -host %s -interface %s",
					state.GatewayIP, state.PhysIfaceName,
				); err != nil {
					n.logger.Debug().
						Err(err).
						Str("output", out).
						Msg("route -n delete -host (ignored)")
				}
				return nil
			},
		},
		{
			Set: func() error {
				for _, target := range state.RouteTargetCIDRs {
					if out, err := executil.Commandf("route -n add -net %s -interface %s",
						target, state.TUNName,
					); err != nil {
						return fmt.Errorf("failed to add route for %s: %s: %w", target, out, err)
					}
				}

				return nil
			},
			Unset: func() error {
				for _, target := range state.RouteTargetCIDRs {
					if out, err := executil.Commandf("route -n delete -net %s -interface %s",
						target, state.TUNName,
					); err != nil {
						n.logger.Debug().Err(err).Str("output", out).Msg("route delete (ignored)")
					}
				}

				return nil
			},
		},
	}
}

func (n *tunSystemNetworkDarwin) Configure(jobs []configurationJob) (func(), error) {
	if staleState, exists, err := loadState(); err == nil && exists {
		n.logger.Info().Str("iface", staleState.TUNName).Msg("cleaning up stale state")

		// Reverts only the successfully applied jobs in LIFO order
		for i := len(jobs) - 1; i >= 0; i-- {
			if err := jobs[i].Unset(); err != nil {
				n.logger.Error().Err(err).Msg("failed to run unset job")
			}
		}

		// Cleans up the persisted state file to ensure complete rollback and teardown.
		if err := deleteState(); err != nil {
			n.logger.Error().Err(err).Msg("failed to delete stale state")
		}
	}

	newState, err := n.createState()
	if err != nil {
		return nil, err
	}

	if err := saveState(newState); err != nil {
		return nil, fmt.Errorf("failed to save state: %w", err)
	}

	// Tracks the count of successfully executed configuration jobs
	var executedJobs int

	unset := func() {
		// Reverts only the successfully applied jobs in LIFO order
		for i := executedJobs - 1; i >= 0; i-- {
			if err := jobs[i].Unset(); err != nil {
				n.logger.Error().Err(err).Msg("failed to run unset job")
			}
		}

		// Cleans up the persisted state file to ensure complete rollback and teardown.
		if err := deleteState(); err != nil {
			n.logger.Error().Err(err).Msg("failed to delete state file during cleanup")
		}
	}

	set := func() error {
		for i, each := range jobs {
			if err := each.Set(); err != nil {
				return fmt.Errorf("failed to run set job: %w", err)
			}
			// Increments the counter after successful job execution
			executedJobs = i + 1 // We use index + 1 in cases none of the job was successfull
		}
		return nil
	}

	if err := set(); err != nil {
		unset()
		return nil, err
	}

	return unset, nil
}
