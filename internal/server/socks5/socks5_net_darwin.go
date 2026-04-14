//go:build darwin

package socks5

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/rs/zerolog"
	"github.com/xvzc/spoofdpi/internal/executil"
	"github.com/xvzc/spoofdpi/internal/netutil"
	"github.com/xvzc/spoofdpi/internal/server"
)

const socks5StateFile = "/tmp/spoofdpi.socks5.darwin.state"

type socks5StateDarwin struct {
	Service    string `json:"service"`
	ServerPort uint16 `json:"serverPort"`
	ProxyType  string `json:"proxyType"`
	PACURL     string `json:"pacURL"`
}

func getNetworkServiceFromInterface(ifaceName string) (string, error) {
	out, err := executil.Commandf("networksetup", "-listnetworkserviceorder")
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(
		fmt.Sprintf(`\(\d+\)\s+(.*)\s+\(Hardware Port:.*Device:\s+%s\)`, ifaceName),
	)
	match := re.FindStringSubmatch(string(out))

	if len(match) < 2 {
		return "", fmt.Errorf("no network service found for interface: %s", ifaceName)
	}

	return strings.TrimSpace(match[1]), nil
}

func createState(
	defaultRoute *netutil.Route, serverPort uint16, pacURL string,
) (*socks5StateDarwin, error) {
	ifaceName := defaultRoute.Iface.Name
	service, err := getNetworkServiceFromInterface(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get network service: %w", err)
	}

	return &socks5StateDarwin{
		Service:    service,
		ServerPort: serverPort,
		ProxyType:  "SOCKS5",
		PACURL:     pacURL,
	}, nil
}

func saveState(state *socks5StateDarwin) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(socks5StateFile, data, 0o644)
}

func loadState() (*socks5StateDarwin, bool, error) {
	data, err := os.ReadFile(socks5StateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	var state socks5StateDarwin
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, false, err
	}
	return &state, true, nil
}

func deleteState() error {
	if err := os.Remove(socks5StateFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func configurationJobs(
	ctx context.Context,
	logger zerolog.Logger,
	state *socks5StateDarwin,
) []server.ConfigurationJob {
	var jobs []server.ConfigurationJob

	jobs = append(jobs, server.ConfigurationJob{
		Up: func() error {
			if out, err := executil.Commandf(
				"networksetup -setautoproxyurl %s %s",
				state.Service,
				state.PACURL,
			); err != nil {
				return fmt.Errorf("setting autoproxyurl: %s: %w", out, err)
			}

			if out, err := executil.Commandf(
				"networksetup -setproxyautodiscovery %s on",
				state.Service,
			); err != nil {
				return fmt.Errorf("setting proxyautodiscovery: %s: %w", out, err)
			}

			return nil
		},
		Down: func() error {
			if out, err := executil.Commandf(
				"networksetup -setautoproxystate %s off",
				state.Service,
			); err != nil {
				logger.Trace().
					Err(err).
					Str("out", out).
					Msg("failed to unset autoproxystate (ignored)")
			}

			if out, err := executil.Commandf(
				"networksetup -setproxyautodiscovery %s off",
				state.Service,
			); err != nil {
				logger.Trace().
					Err(err).
					Str("out", out).
					Msg("failed to unset proxyautodiscovery (ignored)")
			}

			return nil
		},
	})

	return jobs
}

type socks5SystemNetworkDarwin struct {
	logger       zerolog.Logger
	defaultRoute *netutil.Route
}

func NewSOCKS5SystemNetwork(
	logger zerolog.Logger,
	defaultRoute *netutil.Route,
) SOCKS5SystemNetwork {
	return &socks5SystemNetworkDarwin{
		logger:       logger,
		defaultRoute: defaultRoute,
	}
}

func (n *socks5SystemNetworkDarwin) DefaultRoute() *netutil.Route {
	return n.defaultRoute
}
