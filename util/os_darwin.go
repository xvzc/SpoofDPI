//go:build darwin
// +build darwin

package util

import (
	"fmt"
	"os/exec"
	"strings"
)

func SetOsProxy(port int) error {
	network, err := exec.Command("sh", "-c", "networksetup -listnetworkserviceorder | grep `route -n get 0.0.0.0 | grep 'interface' | cut -d ':' -f2` -B 1 | head -n 1 | cut -d ' ' -f 2-").Output()

	if err != nil {
		return err
	}

	_, err = exec.Command("sh", "-c", "networksetup -setwebproxy "+"'"+strings.TrimSpace(string(network))+"'"+" 127.0.0.1 "+fmt.Sprint(port)).Output()
	if err != nil {
		return err
	}

	_, err = exec.Command("sh", "-c", "networksetup -setsecurewebproxy "+"'"+strings.TrimSpace(string(network))+"'"+" 127.0.0.1 "+fmt.Sprint(port)).Output()
	if err != nil {
		return err
	}

	return nil
}

func UnsetOsProxy() error {
	network, err := exec.Command("sh", "-c", "networksetup -listnetworkserviceorder | grep `route -n get 0.0.0.0 | grep 'interface' | cut -d ':' -f2` -B 1 | head -n 1 | cut -d ' ' -f 2-").Output()
	if err != nil {
		return err
	}

	_, err = exec.Command("sh", "-c", "networksetup -setwebproxystate "+"'"+strings.TrimSpace(string(network))+"'"+" off").Output()
	if err != nil {
		return err
	}

	_, err = exec.Command("sh", "-c", "networksetup -setsecurewebproxystate "+"'"+strings.TrimSpace(string(network))+"'"+" off").Output()
	if err != nil {
		return err
	}

	return nil
}
