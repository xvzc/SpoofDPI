//go:build linux

package http

import "github.com/rs/zerolog"

func SetSystemProxy(logger zerolog.Logger, port uint16) error {
	// Not implemented for Linux yet
	return nil
}

func UnsetSystemProxy(logger zerolog.Logger) error {
	return nil
}
