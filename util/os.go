//go:build !windows && !darwin
// +build !windows,!darwin

package util

func SetOsProxy(port int) error {
	return nil
}

func UnsetOsProxy() error {
	return nil
}
