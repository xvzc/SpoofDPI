package util

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

const (
	getDefaultNetworkCMD = "networksetup -listnetworkserviceorder | grep" +
		" `(route -n get default | grep 'interface' || route -n get -inet6 default | grep 'interface') | cut -d ':' -f2`" +
		" -B 1 | head -n 1 | cut -d ' ' -f 2-"
	darwinOS = "darwin"
)

func SetOsProxy(port uint16) error {
	if runtime.GOOS != darwinOS {
		return nil
	}

	network, err := getDefaultNetwork()
	if err != nil {
		return fmt.Errorf("failed to get default network: %w", err)
	}

	return setProxy(getProxyTypes(), network, "127.0.0.1", port)
}

func UnsetOsProxy() error {
	if runtime.GOOS != darwinOS {
		return nil
	}

	network, err := getDefaultNetwork()
	if err != nil {
		return fmt.Errorf("failed to get default network: %w", err)
	}

	return unsetProxy(getProxyTypes(), network)
}

func getDefaultNetwork() (string, error) {
	cmd := exec.Command("sh", "-c", getDefaultNetworkCMD)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %s", cmd.String(), out)
	}
	if len(out) == 0 {
		return "", fmt.Errorf("%s: no available networks", cmd.String())
	}
	return strings.TrimSpace(string(out)), nil
}

func getProxyTypes() []string {
	return []string{"webproxy", "securewebproxy"}
}

func setProxy(proxyTypes []string, network, domain string, port uint16) error {
	args := []string{"", network, domain, strconv.FormatUint(uint64(port), 10)}

	for _, proxyType := range proxyTypes {
		args[0] = "-set" + proxyType
		if err := networkSetup(args); err != nil {
			return fmt.Errorf("setting %s: %w", proxyType, err)
		}
	}
	return nil
}

func unsetProxy(proxyTypes []string, network string) error {
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
		return fmt.Errorf("%s: %s", cmd.String(), out)
	}
	return nil
}
