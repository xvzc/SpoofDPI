package util

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

const getDefaultNetworkCMD = "networksetup -listnetworkserviceorder | grep" +
	" `(route -n get default | grep 'interface' || route -n get -inet6 default | grep 'interface') | cut -d ':' -f2`" +
	" -B 1 | head -n 1 | cut -d ' ' -f 2-"

func SetOsProxy(port int) error {
	if runtime.GOOS != "darwin" {
		return nil
	}

	network, err := getDefaultNetwork()
	if err != nil {
		return fmt.Errorf("failed to get network interfaces, stdout: %s: %w", string(network), err)
	}


	args := fmt.Sprintf("'%s' 127.0.0.1 %d", network, port)

	out, err = exec.Command("sh", "-c", "networksetup -setwebproxy "+args).Output()
	if err != nil {
		return fmt.Errorf("failed to set web proxy, stdout: %s: %w", string(out), err)
	}


	out, err = exec.Command("sh", "-c", "networksetup -setsecurewebproxy "+args).Output()
	if err != nil {
		return fmt.Errorf("failed to set secure web proxy, stdout: %s: %w", string(out), err)
	}

	return nil
}

func UnsetOsProxy() error {
	if runtime.GOOS != "darwin" {
		return nil
	}

	network, err := getDefaultNetwork()
	if err != nil {
		return fmt.Errorf("failed to get network interfaces, stdout: %s: %w", string(network), err)
	}

	out, err = exec.Command("sh", "-c", "networksetup -setwebproxystate "+"'"+network+"'"+" off").Output()
	if err != nil {
		return fmt.Errorf("failed to set web proxy, stdout: %s: %w", string(out), err)
	}

	out, err = exec.Command("sh", "-c", "networksetup -setsecurewebproxystate "+"'"+network+"'"+" off").Output()
	if err != nil {
		return fmt.Errorf("failed to set secure web proxy, stdout: %s: %w", string(out), err)
	}

	return nil
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
