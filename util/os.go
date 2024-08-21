package util

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

func SetOsProxy(port int) error {
	if runtime.GOOS != "darwin" {
		return nil
	}

	network, err := exec.Command("sh", "-c", "networksetup -listnetworkserviceorder | grep `route -n get 0.0.0.0 | grep 'interface' | cut -d ':' -f2` -B 1 | head -n 1 | cut -d ' ' -f 2-").Output()
	if err != nil {
		return fmt.Errorf("failed to get network interfaces, stdout: %s: %w", string(network), err)
	}

	out, err := exec.Command("sh", "-c", "networksetup -setwebproxy "+"'"+strings.TrimSpace(string(network))+"'"+" 127.0.0.1 "+fmt.Sprint(port)).Output()
	if err != nil {
		return fmt.Errorf("failed to set web proxy, stdout: %s: %w", string(out), err)
	}

	out, err = exec.Command("sh", "-c", "networksetup -setsecurewebproxy "+"'"+strings.TrimSpace(string(network))+"'"+" 127.0.0.1 "+fmt.Sprint(port)).Output()
	if err != nil {
		return fmt.Errorf("failed to set secure web proxy, stdout: %s: %w", string(out), err)
	}

	return nil
}

func UnsetOsProxy() error {
	if runtime.GOOS != "darwin" {
		return nil
	}

	network, err := exec.Command("sh", "-c", "networksetup -listnetworkserviceorder | grep `route -n get 0.0.0.0 | grep 'interface' | cut -d ':' -f2` -B 1 | head -n 1 | cut -d ' ' -f 2-").Output()
	if err != nil {
		return fmt.Errorf("failed to get network interfaces, stdout: %s: %w", string(network), err)
	}

	out, err := exec.Command("sh", "-c", "networksetup -setwebproxystate "+"'"+strings.TrimSpace(string(network))+"'"+" off").Output()
	if err != nil {
		return fmt.Errorf("failed to set web proxy, stdout: %s: %w", string(out), err)
	}

	out, err = exec.Command("sh", "-c", "networksetup -setsecurewebproxystate "+"'"+strings.TrimSpace(string(network))+"'"+" off").Output()
	if err != nil {
		return fmt.Errorf("failed to set secure web proxy, stdout: %s: %w", string(out), err)
	}

	return nil
}
