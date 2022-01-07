package config

import (
	"runtime"
	"sync"
)

type Config struct {
	Port  string
	DNS   string
	OS    string
	Debug bool
}

var config *Config
var once sync.Once
var err error

func InitConfig(port string, dns string, debug bool) error {
	err = nil

	once.Do(func() {

		config = &Config{
			Port:  port,
			DNS:   dns,
			OS:    runtime.GOOS,
			Debug: debug,
		}
	})

	return err
}

func GetConfig() *Config {
	return config
}
