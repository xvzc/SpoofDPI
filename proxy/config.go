package proxy

import (
	"errors"
	"log"
	"strings"
    "fmt"
    "sync"
)

type Config struct {
    SrcIp string
    SrcPort string
    DNS string
    MTU int
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

func InitConfig(srcAddress string, dns string, mtu int, debug bool) error {
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
            MTU : mtu,
            Debug : debug,
        }
    })

    log.Println("source ip   : " + config.SrcIp)
    log.Println("source port : " + config.SrcPort)
    log.Println("dns         : " + config.DNS)
    log.Println("mtu         : " + fmt.Sprint(config.MTU))
    if config.Debug {
        log.Println("debug       : true")
    } else {
        log.Println("debug       : false")
    }

    return err
}

func getConfig() (*Config) {
    return config
}
