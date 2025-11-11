package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

func parseTomlConfig(dir string) (*Config, error) {
	var cfg *Config
	_, err := toml.DecodeFile(dir, &cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func findConfigFileToLoad(customDir string, lookupDirs []string) (string, error) {
	if customDir != "" {
		_, err := os.Stat(customDir)
		if err != nil { // Path exists
			return "", fmt.Errorf("no such file: %s", customDir)
		} else {
			return customDir, nil
		}
	}

	for _, p := range lookupDirs {
		if p == "" {
			continue
		}

		if _, err := os.Stat(p); err == nil { // Path exists
			return p, nil
		}
	}

	// Don't care even if config files are not found in lookupDirs
	return "", nil
}
