package config

import (
	"os/exec"
	"strings"
)


func SetOsProxy() error {
    if GetConfig().OS != "darwin" {
        return nil
    }

    network, err:= exec.Command("sh", "-c", "networksetup -listnetworkserviceorder | grep `route -n get 0.0.0.0 | grep 'interface' | cut -d ':' -f2` -B 1 | head -n 1 | cut -d ' ' -f2").Output()
    if err != nil {
        return err
    }

    _, err = exec.Command("sh", "-c", "networksetup -setwebproxy " + strings.TrimSpace(string(network)) + " 127.0.0.1 " + GetConfig().Port).Output()
    if err != nil {
        return err
    }

    _, err = exec.Command("sh", "-c", "networksetup -setsecurewebproxy " + strings.TrimSpace(string(network)) + " 127.0.0.1 " + GetConfig().Port).Output()
    if err != nil {
        return err
    }

    return nil
}
