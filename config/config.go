package config

import (
	"log"
	"regexp"
	"sync"
)

type Config struct {
	addr            string
	port            int
	dnsAddr         string
	dnsPort         int
	dnsIPv4Only     bool
	enableDoh       bool
	debug           bool
	silent          bool
	setSystemProxy  bool
	timeout         int
	windowSize      int
	allowedPatterns []*regexp.Regexp
}

var lock = &sync.Mutex{}
var config *Config

func Get() *Config {
	if config == nil {
		log.Fatal("Config is not loaded.")
	}

	return config
}

func New(args *Args) *Config {
	if config == nil {
		lock.Lock()
		defer lock.Unlock()
		if config == nil {
			config = &Config{
				addr:            args.Addr,
				allowedPatterns: parseAllowedPatterns(args.AllowedPattern),
				debug:           args.Debug,
				dnsAddr:         args.DnsAddr,
				dnsPort:         int(args.DnsPort),
				dnsIPv4Only:     args.DnsIPv4Only,
				enableDoh:       args.EnableDoh,
				setSystemProxy:  args.SystemProxy,
				port:            int(args.Port),
				silent:          args.Silent,
				timeout:         int(args.Timeout),
				windowSize:      int(args.WindowSize),
			}
		}
	}

	return config
}

func (c *Config) Addr() string {
	return c.addr
}

func (c *Config) AllowedPatterns() []*regexp.Regexp {
	return c.allowedPatterns
}

func (c *Config) Debug() bool {
	return c.debug
}

func (c *Config) DnsAddr() string {
	return c.dnsAddr
}

func (c *Config) DnsPort() int {
	return c.dnsPort
}

func (c *Config) DnsIPv4Only() bool {
	return c.dnsIPv4Only
}

func (c *Config) EnableDoh() bool {
	return c.enableDoh
}

func (c *Config) Port() int {
	return c.port
}

func (c *Config) SetSystemProxy() bool {
	return c.setSystemProxy
}

func (c *Config) Silent() bool {
	return c.silent
}

func (c *Config) Timeout() int {
	return c.timeout
}

func (c *Config) WindowSize() int {
	return c.windowSize
}
