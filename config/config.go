package config 

import (
	"errors"
	"log"
	"strings"
    "sync"
    "runtime"
)

type Config struct {
    SrcIp string
    SrcPort string
    DNS string
    OS string
    Debug bool
}

var config *Config
var once sync.Once
var err error

func tokenizeAddress(srcAddress string) (string, string, error) {
    tokens := strings.Split(srcAddress, ":")
    if len(tokens) < 2 {
        return "", "", errors.New("Error while parsing source address: invalid format.")
    }

    ip := tokens[0]
    port := tokens[1]

    return ip, port, nil
}

func InitConfig(srcAddress string, dns string, debug bool) error {
    err = nil

    once.Do(func() {
        ip, port, err := tokenizeAddress(srcAddress)
        if err != nil {
            log.Fatal(err)
            return
        }

        config = &Config{
            SrcIp : ip,
            SrcPort : port,
            DNS : dns,
            OS : runtime.GOOS,
            Debug : debug,
        }
    })

    return err
}

func GetConfig() (*Config) {
    return config
}
