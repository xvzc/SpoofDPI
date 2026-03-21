//go:build darwin

package socks5

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/netutil"
)

const (
	permissionErrorHelpText = "By default SpoofDPI tries to set itself up as a system-wide proxy server.\n" +
		"Doing so may require root access on machines with\n" +
		"'Settings > Privacy & Security > Advanced > Require" +
		" an administrator password to access system-wide settings' enabled.\n" +
		"If you do not want SpoofDPI to act as a system-wide proxy, provide" +
		" -system-proxy=false."
)

func setSystemProxy(logger zerolog.Logger, port uint16) (func() error, error) {
	network, err := getDefaultNetwork()
	if err != nil {
		return nil, err
	}

	portStr := strconv.Itoa(int(port))
	pacContent := fmt.Sprintf(`function FindProxyForURL(url, host) {
    return "SOCKS5 127.0.0.1:%s; DIRECT";
}`, portStr)

	pacURL, pacServer, err := netutil.RunPACServer(pacContent)
	if err != nil {
		return nil, fmt.Errorf("error creating pac server: %w", err)
	}

	// Enable Auto Proxy Configuration
	// networksetup -setautoproxyurl <networkservice> <url>
	if err := networkSetup("-setautoproxyurl", network, pacURL); err != nil {
		_ = pacServer.Close()
		return nil, fmt.Errorf("setting autoproxyurl: %w", err)
	}

	// networksetup -setproxyautodiscovery <networkservice> <on off>
	if err := networkSetup("-setproxyautodiscovery", network, "on"); err != nil {
		_ = pacServer.Close()
		return nil, fmt.Errorf("setting proxyautodiscovery: %w", err)
	}

	unset := func() error {
		_ = pacServer.Close()

		if err := networkSetup("-setautoproxystate", network, "off"); err != nil {
			return fmt.Errorf("unsetting autoproxystate: %w", err)
		}

		if err := networkSetup("-setproxyautodiscovery", network, "off"); err != nil {
			return fmt.Errorf("unsetting proxyautodiscovery: %w", err)
		}

		return nil
	}

	return unset, nil
}

func getDefaultNetwork() (string, error) {
	const cmd = "networksetup -listnetworkserviceorder | grep" +
		" `(route -n get default | grep 'interface' || route -n get -inet6 default | grep 'interface') | cut -d ':' -f2`" +
		" -B 1 | head -n 1 | cut -d ' ' -f 2-"

	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		return "", err
	}

	network := strings.TrimSpace(string(out))
	if network == "" {
		return "", errors.New("no available networks")
	}
	return network, nil
}

func networkSetup(args ...string) error {
	cmd := exec.Command("networksetup", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := string(out)
		if isPermissionError(err) {
			msg += permissionErrorHelpText
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func isPermissionError(err error) bool {
	var exitErr *exec.ExitError
	ok := errors.As(err, &exitErr)
	return ok && exitErr.ExitCode() == 14
}
