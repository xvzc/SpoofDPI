//go:build windows
// +build windows

package util

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

func SetOsProxy(port int) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	err = key.SetDWordValue("ProxyEnable", 1)
	if err != nil {
		return err
	}

	err = key.SetStringValue("ProxyServer", "127.0.0.1:"+fmt.Sprint(port))
	if err != nil {
		return err
	}

	return nil
}

func UnsetOsProxy() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	err = key.SetDWordValue("ProxyEnable", 0)
	if err != nil {
		return err
	}

	return nil
}
