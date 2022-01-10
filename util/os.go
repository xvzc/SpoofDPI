package util

import (
	"os/exec"
	"runtime"
	"strings"
)

func SetOsProxy(port string) error {
	if runtime.GOOS != "darwin" {
		return nil
	}

	network, err := exec.Command("sh", "-c", "networksetup -listnetworkserviceorder | grep `route -n get 0.0.0.0 | grep 'interface' | cut -d ':' -f2` -B 1 | head -n 1 | cut -d ' ' -f2").Output()
	if err != nil {
		return err
	}

	_, err = exec.Command("sh", "-c", "networksetup -setwebproxy "+strings.TrimSpace(string(network))+" 127.0.0.1 "+port).Output()
	if err != nil {
		return err
	}

	_, err = exec.Command("sh", "-c", "networksetup -setsecurewebproxy "+strings.TrimSpace(string(network))+" 127.0.0.1 "+port).Output()
	if err != nil {
		return err
	}

	return nil
}

func UnsetOsProxy() error {
	if runtime.GOOS != "darwin" {
		return nil
	}

	network, err := exec.Command("sh", "-c", "networksetup -listnetworkserviceorder | grep `route -n get 0.0.0.0 | grep 'interface' | cut -d ':' -f2` -B 1 | head -n 1 | cut -d ' ' -f2").Output()
	if err != nil {
		return err
	}

	_, err = exec.Command("sh", "-c", "networksetup -setwebproxystate "+strings.TrimSpace(string(network))+" off").Output()
	if err != nil {
		return err
	}

	_, err = exec.Command("sh", "-c", "networksetup -setsecurewebproxystate "+strings.TrimSpace(string(network))+" off").Output()
	if err != nil {
		return err
	}

	return nil
}
