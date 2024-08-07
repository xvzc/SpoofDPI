package util

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

func SetOsProxy(port int) error {
	if runtime.GOOS == "darwin" {
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
	} else if runtime.GOOS == "linux" {
		// dconf write /system/proxy/http/port "8080"
		// dconf write /system/proxy/http/host "'127.0.0.1'"
		// dconf write /system/proxy/mode "'manual'"
		var err error

		_, err = exec.Command("sh", "-c", "dconf write /system/proxy/http/port \"" + fmt.Sprint(port) + "\"").Output()
		if err != nil {
			return err
		}

		_, err = exec.Command("sh", "-c", "dconf write /system/proxy/http/host \"'127.0.0.1'\"").Output()
		if err != nil {
			return err
		}

		_, err = exec.Command("sh", "-c", "dconf write /system/proxy/mode \"'manual'\"").Output()
		if err != nil {
			return err
		}
	} else {
		// TO-DO: Output a message to the INFO log that the system-wide proxy could not be set because the OS is not supported
		return nil
	}

	return nil
}

func UnsetOsProxy() error {
	if runtime.GOOS == "darwin" {
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
	} else if runtime.GOOS == "linux" {
		var err error

		_, err = exec.Command("sh", "-c", "dconf write /system/proxy/mode \"'none'\"").Output()
		if err != nil {
			return err
		}
	} else {
		// TO-DO: Output a message to the INFO log that the system-wide proxy could not be unset because the OS is not supported
		return nil
	}

	return nil
}
