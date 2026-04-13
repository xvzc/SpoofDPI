//go:build freebsd

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
	"github.com/xvzc/spoofdpi/internal/server"
	"golang.zx2c4.com/wireguard/tun"
)

const stateFile = "/tmp/spoofdpi.freebsd.tun.state"

func createTunDevice() (tun.Device, error) {
	return tun.CreateTUN("tun%d", 1500)
}

func createState(sysNet TUNSystemNetwork) (*tunStateFreeBSD, error) {
	tunName, err := sysNet.TunDevice().Name()
	if err != nil {
		return nil, fmt.Errorf("failed to get tunName: %w", err)
	}

	subnet, err := getInterfaceSubnet(sysNet.DefaultRoute().Iface.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get interface subnet: %w", err)
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

	state := &tunStateFreeBSD{
		FIBID:            sysNet.FIBID(),
		Gateway:          sysNet.DefaultRoute().Gateway.String(),
		PhysIfaceName:    sysNet.DefaultRoute().Iface.Name,
		Subnet:           subnet,
		TUNName:          tunName,
		TunLocalIP:       tunLocalIP,
		TunRemoteIP:      tunRemoteIP,
		RouteTargetCIDRs: routeTargetCIDRS,
		CreatedAt:        time.Now(),
	}
	return state, nil
}

func saveState(state *tunStateFreeBSD) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(stateFile, data, 0o644)
}

func loadState() (*tunStateFreeBSD, bool, error) {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	var state tunStateFreeBSD
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

type tunStateFreeBSD struct {
	FIBID            int       `json:"fibID"`
	Gateway          string    `json:"gateway"`
	PhysIfaceName    string    `json:"physIfaceName"`
	Subnet           string    `json:"subnet"`
	TUNName          string    `json:"tunName"`
	TunLocalIP       string    `json:"tunLocalIP"`
	TunRemoteIP      string    `json:"tunRemoteIP"`
	RouteTargetCIDRs []string  `json:"routeTargetCIDRs"`
	CreatedAt        time.Time `json:"createdAt"`
}

type tunSystemNetworkFreeBSD struct {
	logger       zerolog.Logger
	tunDevice    tun.Device
	defaultRoute *netutil.Route
	fibID        int
}

func NewTUNSystemNetwork(
	logger zerolog.Logger,
	defaultRoute *netutil.Route,
	fibID int,
) (TUNSystemNetwork, error) {
	dev, err := createTunDevice()
	if err != nil {
		return nil, err
	}

	return &tunSystemNetworkFreeBSD{
		logger:       logger,
		tunDevice:    dev,
		defaultRoute: defaultRoute,
		fibID:        fibID,
	}, nil
}

func (n *tunSystemNetworkFreeBSD) TunDevice() tun.Device {
	return n.tunDevice
}

func (n *tunSystemNetworkFreeBSD) DefaultRoute() *netutil.Route {
	return n.defaultRoute
}

func (n *tunSystemNetworkFreeBSD) FIBID() int {
	return n.fibID
}

func (n *tunSystemNetworkFreeBSD) BindDialer(
	dialer *net.Dialer,
	network string,
	targetIP net.IP,
) error {
	if n.fibID <= 0 || n.defaultRoute == nil || n.defaultRoute.Iface.Name == "" {
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

func configurationJobs(
	logger zerolog.Logger,
	state *tunStateFreeBSD,
) []server.ConfigurationJob {
	var jobs []server.ConfigurationJob

	jobs = append(jobs, server.ConfigurationJob{
		Up: func() error {
			if _, err := executil.Commandf("setfib %d route get default",
				state.FIBID,
			); err == nil {
				return fmt.Errorf("FIB %d is already in use", state.FIBID)
			}
			return nil
		},
		Down: nil,
	})

	jobs = append(jobs, server.ConfigurationJob{
		Up: func() error {
			if out, err := executil.Commandf("ifconfig %s %s %s up",
				state.TUNName, state.TunLocalIP, state.TunRemoteIP,
			); err != nil {
				return fmt.Errorf("failed to set interface address: %s: %w", out, err)
			}
			return nil
		},
		Down: func() error {
			if out, err := executil.Commandf("ifconfig %s destroy",
				state.TUNName,
			); err != nil {
				logger.Debug().Err(err).Str("out", out).Msg("ifconfig destroy (ignored)")
			}
			return nil
		},
	})

	jobs = append(jobs, server.ConfigurationJob{
		Up: func() error {
			if out, err := executil.Commandf("route add -net %s -iface %s -fib %d",
				state.Subnet, state.PhysIfaceName, state.FIBID,
			); err != nil {
				if !strings.Contains(out, "File exists") {
					return fmt.Errorf("failed to add FIB subnet route: %s: %w", out, err)
				}
			}
			return nil
		},
		Down: func() error {
			if out, err := executil.Commandf("route delete -net %s -iface %s -fib %d",
				state.Subnet, state.PhysIfaceName, state.FIBID,
			); err != nil {
				if !strings.Contains(out, "not in table") {
					logger.Debug().
						Err(err).
						Str("out", out).
						Msg("route delete subnet (ignored)")
				}
			}
			return nil
		},
	})

	jobs = append(jobs, server.ConfigurationJob{
		Up: func() error {
			if out, err := executil.Commandf("route add default %s -fib %d",
				state.Gateway, state.FIBID,
			); err != nil {
				if !strings.Contains(out, "File exists") {
					return fmt.Errorf("failed to add FIB default route: %s: %w", out, err)
				}
			}
			return nil
		},
		Down: func() error {
			if out, err := executil.Commandf("route delete default -fib %d",
				state.FIBID); err != nil {
				if !strings.Contains(out, "not in table") {
					logger.Debug().
						Err(err).
						Str("out", out).
						Msg("route delete default (ignored)")
				}
			}
			return nil
		},
	})

	for _, target := range state.RouteTargetCIDRs {
		target := target
		jobs = append(jobs, server.ConfigurationJob{
			Up: func() error {
				if out, err := executil.Commandf("route -n add -net %s -interface %s",
					target, state.TUNName); err != nil {
					if strings.Contains(out, "must be root") {
						return fmt.Errorf(
							"permission denied: must run as root to modify routing table",
						)
					}
					if !strings.Contains(out, "File exists") {
						return fmt.Errorf("failed to add route for %s: %s: %w", target, out, err)
					}
				}
				return nil
			},
			Down: func() error {
				if out, err := executil.Commandf("route -n delete -net %s -interface %s",
					target, state.TUNName); err != nil {
					logger.Debug().
						Err(err).
						Str("out", out).
						Msg("route -n delete (ignored)")
				}
				return nil
			},
		})
	}

	return jobs
}

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
