//go:build !windows && !darwin
// +build !windows,!darwin

package system

func SetProxy(port int) error {
	return nil
}

func UnsetProxy() error {
	return nil
}
