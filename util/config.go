package util

import (
	"flag"
	"fmt"
	"regexp"

	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
)

type Config struct {
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

var config *Config

func GetConfig() *Config {
	return config
}

func ParseArgs() {
	config = &Config{}
	config.Addr = flag.String("addr", "127.0.0.1", "listen address")
	config.Port = flag.Int("port", 8080, "port")
	config.DnsAddr = flag.String("dns-addr", "8.8.8.8", "dns address")
	config.DnsPort = flag.Int("dns-port", 53, "port number for dns")
	config.EnableDoh = flag.Bool("enable-doh", false, "enable 'dns-over-https'")
	config.Debug = flag.Bool("debug", false, "enable debug output")
	config.NoBanner = flag.Bool("no-banner", false, "disable banner")
	config.SystemProxy = flag.Bool("system-proxy", true, "enable system-wide proxy")
	config.Timeout = flag.Int("timeout", 0, "timeout in milliseconds; no timeout when not given")
	config.WindowSize = flag.Int("window-size", 0, `chunk size, in number of bytes, for fragmented client hello,
try lower values if the default value doesn't bypass the DPI;
when not given, the client hello packet will be sent in two parts:
fragmentation for the first data packet and the rest
`)
	config.Version = flag.Bool("v", false, "print spoof-dpi's version; this may contain some other relevant information")

	var allowedPattern StringArray
	flag.Var(
		&allowedPattern,
		"pattern",
		"bypass DPI only on packets matching this regex pattern; can be given multiple times",
	)
	flag.Parse()

	for _, pattern := range allowedPattern {
		config.AllowedPattern = append(config.AllowedPattern, regexp.MustCompile(pattern))
	}
}

func PrintColoredBanner() {
	cyan := putils.LettersFromStringWithStyle("Spoof", pterm.NewStyle(pterm.FgCyan))
	purple := putils.LettersFromStringWithStyle("DPI", pterm.NewStyle(pterm.FgLightMagenta))
	pterm.DefaultBigText.WithLetters(cyan, purple).Render()

	pterm.DefaultBulletList.WithItems([]pterm.BulletListItem{
		{Level: 0, Text: "ADDR    : " + fmt.Sprint(*config.Addr)},
		{Level: 0, Text: "PORT    : " + fmt.Sprint(*config.Port)},
		{Level: 0, Text: "DNS     : " + fmt.Sprint(*config.DnsAddr)},
		{Level: 0, Text: "DEBUG   : " + fmt.Sprint(*config.Debug)},
	}).Render()
}

func PrintSimpleInfo() {
	fmt.Println("")
	fmt.Println("- ADDR    : ", *config.Addr)
	fmt.Println("- PORT    : ", *config.Port)
	fmt.Println("- DNS     : ", *config.DnsAddr)
	fmt.Println("- DEBUG   : ", *config.Debug)
	fmt.Println("")
}
