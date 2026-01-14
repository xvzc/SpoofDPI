//go:build !darwin && !linux

package socks5

import "github.com/rs/zerolog"

func SetSystemProxy(logger zerolog.Logger, port uint16) error {
	return nil
}

func UnsetSystemProxy(logger zerolog.Logger) error {
	return nil
}
