package util

import (
	"flag"
	"fmt"
	"regexp"
	"strings"

	"github.com/pterm/pterm"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	Addr           *string
	Port           *int
	Dns            *string
	Debug          *bool
	NoBanner *bool
	Timeout        *int
	AllowedPattern *regexp.Regexp
	AllowedUrls    *regexp.Regexp
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

func (c *Config) PatternExists() bool {
	return c.AllowedPattern != nil || c.AllowedUrls != nil
}

func (c *Config) PatternMatches(bytes []byte) bool {
	return (c.AllowedPattern != nil && c.AllowedPattern.Match(bytes)) ||
		(c.AllowedUrls != nil && c.AllowedUrls.Match(bytes))
}

func ParseArgs() {
	config = &Config{}
	config.Addr = flag.String("addr", "127.0.0.1", "Listen addr")
	config.Port = flag.Int("port", 8080, "port")
	config.Dns = flag.String("dns", "8.8.8.8", "DNS server")
	config.Debug = flag.Bool("debug", false, "true | false")
	config.NoBanner = flag.Bool("no-banner", false, "true | false")
	config.Timeout = flag.Int("timeout", 2000, "timeout in milliseconds")

	flag.Var(&allowedHosts, "url", "Bypass DPI only on this url, can be passed multiple times")
	allowedPattern = flag.String(
		"pattern",
		"",
		"Bypass DPI only on packets matching this regex pattern",
	)

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
	cyan := pterm.NewLettersFromStringWithStyle("Spoof", pterm.NewStyle(pterm.FgCyan))
	purple := pterm.NewLettersFromStringWithStyle("DPI", pterm.NewStyle(pterm.FgLightMagenta))
	pterm.DefaultBigText.WithLetters(cyan, purple).Render()

	pterm.DefaultBulletList.WithItems([]pterm.BulletListItem{
		{Level: 0, Text: "ADDR    : " + fmt.Sprint(*config.Addr)},
		{Level: 0, Text: "PORT    : " + fmt.Sprint(*config.Port)},
		{Level: 0, Text: "DNS     : " + fmt.Sprint(*config.Dns)},
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
	fmt.Println("- DNS     : ", *config.Dns)
	fmt.Println("- DEBUG   : ", *config.Debug)
	fmt.Println("")
}
