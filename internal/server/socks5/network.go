//go:build !darwin

package socks5

import (
	"github.com/rs/zerolog"
)

func setSystemProxy(logger zerolog.Logger, port uint16) (func() error, error) {
	return func() error { return nil }, nil
}
