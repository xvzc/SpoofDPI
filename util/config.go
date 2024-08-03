package util

import (
	"flag"
	"fmt"
	"regexp"
	"strings"

	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	Addr           *string
	Port           *int
	DnsAddr        *string
	DnsPort        *int
	EnableDoh      *bool
	Debug          *bool
	NoBanner       *bool
	Timeout        *int
	AllowedPattern *regexp.Regexp
	AllowedUrls    *regexp.Regexp
	WindowSize     *int
	Version        *bool
}

type ArrayFlags []string

func (i *ArrayFlags) String() string {
	return "my string representation"
}

func (i *ArrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var config *Config
var allowedHosts ArrayFlags
var allowedPattern *string

func GetConfig() *Config {
	return config
}

func ParseArgs() {
	config = &Config{}
	config.Addr = flag.String("addr", "127.0.0.1", "listen address")
	config.Port = flag.Int("port", 8080, "port")
	config.DnsAddr = flag.String("dns-addr", "8.8.8.8", "dns address")
	config.DnsPort = flag.Int("dns-port", 53, "port number for dns")
	config.EnableDoh = flag.Bool("enable-doh", false, "enable 'dns over https'")
	config.Debug = flag.Bool("debug", false, "enable debug output")
	config.NoBanner = flag.Bool("no-banner", false, "disable banner")
	config.Timeout = flag.Int("timeout", 0, "timeout in milliseconds. no timeout when not given")
	config.WindowSize = flag.Int("window-size", 0, `chunk size, in number of bytes, for fragmented client hello,
try lower values if the default value doesn't bypass the DPI;
when not given, the client hello packet will be sent in two parts:
fragmentation for the first data packet and the rest
`)
	flag.Var(&allowedHosts, "url", "Bypass DPI only on this url, can be passed multiple times")
	allowedPattern = flag.String(
		"pattern",
		"",
		"bypass DPI only on packets matching this regex pattern",
	)
	config.Version = flag.Bool("v", false, "print spoof-dpi's version. this may contain some other relevant information")

	flag.Parse()

	if len(allowedHosts) > 0 {
		var escapedUrls []string
		for _, host := range allowedHosts {
			escapedUrls = append(escapedUrls, regexp.QuoteMeta(host))
		}

		allowedHostsRegex := strings.Join(escapedUrls, "|")
		config.AllowedUrls = regexp.MustCompile(allowedHostsRegex)
	}

	if *allowedPattern != "" {
		config.AllowedPattern = regexp.MustCompile(*allowedPattern)
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

	if allowedHosts != nil && len(allowedHosts) > 0 {
		log.Info("White listed urls: ", allowedHosts)
	}

	if *allowedPattern != "" {
		log.Info("Regex Pattern: ", *allowedPattern)
	}
}

func PrintSimpleInfo() {
	fmt.Println("")
	fmt.Println("- ADDR    : ", *config.Addr)
	fmt.Println("- PORT    : ", *config.Port)
	fmt.Println("- DNS     : ", *config.DnsAddr)
	fmt.Println("- DEBUG   : ", *config.Debug)
	fmt.Println("")
}
