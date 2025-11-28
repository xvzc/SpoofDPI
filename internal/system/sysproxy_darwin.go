//go:build darwin

package system

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
)

const (
	getDefaultNetworkCMD = "networksetup -listnetworkserviceorder | grep" +
		" `(route -n get default | grep 'interface' || route -n get -inet6 default | grep 'interface') | cut -d ':' -f2`" +
		" -B 1 | head -n 1 | cut -d ' ' -f 2-"
	permissionErrorHelpText = "By default SpoofDPI tries to set itself up as a system-wide proxy server.\n" +
		"Doing so may require root access on machines with\n" +
		"'Settings > Privacy & Security > Advanced > Require" +
		" an administrator password to access system-wide settings' enabled.\n" +
		"If you do not want SpoofDPI to act as a system-wide proxy, provide" +
		" -system-proxy=false."
)

func SetProxy(logger zerolog.Logger, port uint16) error {
	network, err := getDefaultNetwork()
	if err != nil {
		return err
	}

	return setProxyInternal(getProxyTypes(), network, "127.0.0.1", int(port))
}

func UnsetProxy(logger zerolog.Logger) error {
	network, err := getDefaultNetwork()
	if err != nil {
		return err
	}

	return unsetProxyInternal(getProxyTypes(), network)
}

func getDefaultNetwork() (string, error) {
	network, err := exec.Command("sh", "-c", getDefaultNetworkCMD).Output()
	if err != nil {
		return "", err
	} else if len(network) == 0 {
		return "", errors.New("no available networks")
	}
	return strings.TrimSpace(string(network)), nil
}

func getProxyTypes() []string {
	return []string{"webproxy", "securewebproxy"}
}

func setProxyInternal(proxyTypes []string, network, domain string, port int) error {
	args := []string{"", network, domain, strconv.FormatUint(uint64(port), 10)}

	for _, proxyType := range proxyTypes {
		args[0] = "-set" + proxyType
		if err := networkSetup(args); err != nil {
			return fmt.Errorf("setting %s: %w", proxyType, err)
		}
	}
	return nil
}

func unsetProxyInternal(proxyTypes []string, network string) error {
	args := []string{"", network, "off"}

	for _, proxyType := range proxyTypes {
		args[0] = "-set" + proxyType + "state"
		if err := networkSetup(args); err != nil {
			return fmt.Errorf("unsetting %s: %w", proxyType, err)
		}
	}
	return nil
}

func networkSetup(args []string) error {
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
