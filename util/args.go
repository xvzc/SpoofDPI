package util

import (
	"flag"
	"fmt"
)

type Args struct {
	Addr           string
	Port           int
	DnsAddr        string
	DnsPort        int
	EnableDoh      bool
	Debug          bool
	Banner         bool
	SystemProxy    bool
	Timeout        int
	AllowedPattern StringArray
	WindowSize     int
	Version        bool
}

type StringArray []string

func (arr *StringArray) String() string {
	return fmt.Sprintf("%s", *arr)
}

func (arr *StringArray) Set(value string) error {
	*arr = append(*arr, value)
	return nil
}

func ParseArgs() *Args {
	args := new(Args)

	flag.StringVar(&args.Addr, "addr", "127.0.0.1", "listen address")
	flag.IntVar(&args.Port, "port", 8080, "port")
	flag.StringVar(&args.DnsAddr, "dns-addr", "8.8.8.8", "dns address")
	flag.IntVar(&args.DnsPort, "dns-port", 53, "port number for dns")
	flag.BoolVar(&args.EnableDoh, "enable-doh", false, "enable 'dns-over-https'")
	flag.BoolVar(&args.Debug, "debug", false, "enable debug output")
	flag.BoolVar(&args.Banner, "banner", true, "enable banner")
	flag.BoolVar(&args.SystemProxy, "system-proxy", true, "enable system-wide proxy")
	flag.IntVar(&args.Timeout, "timeout", 0, "timeout in milliseconds; no timeout when not given")
	flag.IntVar(&args.WindowSize, "window-size", 0, `chunk size, in number of bytes, for fragmented client hello,
try lower values if the default value doesn't bypass the DPI;
when not given, the client hello packet will be sent in two parts:
fragmentation for the first data packet and the rest
`)
	flag.BoolVar(&args.Version, "v", false, "print spoof-dpi's version; this may contain some other relevant information")
	flag.Var(
		&args.AllowedPattern,
		"pattern",
		"bypass DPI only on packets matching this regex pattern; can be given multiple times",
	)

	flag.Parse()

	return args
}
