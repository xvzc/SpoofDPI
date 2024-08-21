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
		return err
	}

	args := fmt.Sprintf("'%s' 127.0.0.1 %d", network, port)

	_, err = exec.Command("sh", "-c", "networksetup -setwebproxy "+args).Output()
	if err != nil {
		return err
	}

	_, err = exec.Command("sh", "-c", "networksetup -setsecurewebproxy "+args).Output()
	if err != nil {
		return err
	}

	return nil
}

func UnsetOsProxy() error {
	if runtime.GOOS != "darwin" {
		return nil
	}

	network, err := getDefaultNetwork()
	if err != nil {
		return err
	}

	_, err = exec.Command("sh", "-c", "networksetup -setwebproxystate "+"'"+network+"'"+" off").Output()
	if err != nil {
		return err
	}

	_, err = exec.Command("sh", "-c", "networksetup -setsecurewebproxystate "+"'"+network+"'"+" off").Output()
	if err != nil {
		return err
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
