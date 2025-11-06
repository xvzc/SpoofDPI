//go:build !windows && !darwin

package system

func SetProxy(port uint16) error {
	return nil
}

func UnsetProxy() error {
	return nil
}
