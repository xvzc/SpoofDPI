package util

import (
	"flag"
	"fmt"
	"regexp"
)

type Args struct {
	Addr           *string
	Port           *int
	DnsAddr        *string
	DnsPort        *int
	EnableDoh      *bool
	Debug          *bool
	NoBanner       *bool
	SystemProxy    *bool
	Timeout        *int
	AllowedPattern []*regexp.Regexp
	WindowSize     *int
	Version        *bool
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
	args.Addr = flag.String("addr", "127.0.0.1", "listen address")
	args.Port = flag.Int("port", 8080, "port")
	args.DnsAddr = flag.String("dns-addr", "8.8.8.8", "dns address")
	args.DnsPort = flag.Int("dns-port", 53, "port number for dns")
	args.EnableDoh = flag.Bool("enable-doh", false, "enable 'dns-over-https'")
	args.Debug = flag.Bool("debug", false, "enable debug output")
	args.NoBanner = flag.Bool("no-banner", false, "disable banner")
	args.SystemProxy = flag.Bool("system-proxy", true, "enable system-wide proxy")
	args.Timeout = flag.Int("timeout", 0, "timeout in milliseconds; no timeout when not given")
	args.WindowSize = flag.Int("window-size", 0, `chunk size, in number of bytes, for fragmented client hello,
try lower values if the default value doesn't bypass the DPI;
when not given, the client hello packet will be sent in two parts:
fragmentation for the first data packet and the rest
`)
	args.Version = flag.Bool("v", false, "print spoof-dpi's version; this may contain some other relevant information")

	var allowedPattern StringArray
	flag.Var(
		&allowedPattern,
		"pattern",
		"bypass DPI only on packets matching this regex pattern; can be given multiple times",
	)

	flag.Parse()

	for _, pattern := range allowedPattern {
		args.AllowedPattern = append(args.AllowedPattern, regexp.MustCompile(pattern))
	}

	return args
}
