//go:build linux

package system

import "github.com/rs/zerolog"

func SetProxy(port uint16, logger zerolog.Logger) error {
	logger.Info().Msgf("automatic system-wide proxy setup is not implemented on Linux")
	return nil
}

func UnsetProxy(logger zerolog.Logger) error {
	return nil
}
