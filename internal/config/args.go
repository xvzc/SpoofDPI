package config

import (
	"flag"
	"fmt"
)

type Args struct {
	AllowedPattern   StringArray
	CacheShards      uint
	Debug            bool
	DnsAddr          string
	DnsPort          uint
	DnsIPv4Only      bool
	EnableDOH        bool
	FakeHTTPSPackets uint
	ListenAddr       string
	ListenPort       uint
	Silent           bool
	SystemProxy      bool
	Timeout          uint
	Version          bool
	WindowSize       uint
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
	flag.UintVar(
		&args.CacheShards,
		"cache-shards",
		32,
		`number of shards to use for ttlcache; it is recommended to set 
this to be >= the number of CPU cores for optimal performance (max 256)`,
	)
	flag.BoolVar(&args.Debug, "debug", false, "enable debug output")
	flag.StringVar(&args.DnsAddr, "dns-addr", "8.8.8.8", "dns address")
	flag.BoolVar(
		&args.DnsIPv4Only,
		"dns-ipv4-only",
		false,
		"resolve only IPv4 addresses",
	)
	flag.UintVar(&args.DnsPort, "dns-port", 53, "port number for dns")
	flag.BoolVar(&args.EnableDOH, "enable-doh", false, "enable 'dns-over-https'")
	flag.UintVar(
		&args.FakeHTTPSPackets,
		"fake-https-packets",
		0,
		`number of fake packets to send before the client hello (default 0, max 50)
higher values may increase success, but the lowest possible value is recommended.
try this if tcp-level fragmentation (via --window-size) does not work.
this feature requires root privilege and the 'libpcap' dependency`,
	)
	flag.StringVar(
		&args.ListenAddr,
		"listen-addr",
		"127.0.0.1",
		"IP address to listen on",
	)
	flag.UintVar(&args.ListenPort, "listen-port", 8080, "port number to listen on")
	flag.Var(
		&args.AllowedPattern,
		"pattern",
		"bypass DPI only on packets matching this regex pattern; can be given multiple times",
	)
	flag.BoolVar(
		&args.Silent,
		"silent",
		false,
		"do not show the banner and server information at start up",
	)
	flag.BoolVar(&args.SystemProxy, "system-proxy", true, "enable system-wide proxy")
	flag.UintVar(
		&args.Timeout,
		"timeout",
		0,
		"timeout in milliseconds; no timeout when not given",
	)
	flag.BoolVar(
		&args.Version,
		"v",
		false,
		"print spoofdpi's version; this may contain some other relevant information",
	)
	flag.UintVar(
		&args.WindowSize,
		"window-size",
		0,
		`chunk size, in number of bytes, for fragmented client hello,
try lower values if the default value doesn't bypass the DPI;
when not given, the client hello packet will be sent in two parts:
fragmentation for the first data packet and the rest
`,
	)

	flag.Parse()

	return args
}
