package util

import (
	"bufio"
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
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
	AllowedPattern patternSet
	PatternFile    *string
	WindowSize     *int
	Version        *bool
}

type patternSet map[*regexp.Regexp]struct{}

func (ps *patternSet) Merge(patterns patternSet) {
	if *ps == nil {
		*ps = patterns
	} else if patterns != nil && len(patterns) > 0 {
		maps.Copy(*ps, patterns)
	}
}

type StringSet map[string]struct{}

func (ps *StringSet) String() string {
	return fmt.Sprintf("%s", *ps)
}

func (ps *StringSet) Set(value string) error {
	(*ps)[value] = struct{}{}
	return nil
}

var config *Config

func (c *Config) Load() error {
	if *c.PatternFile != "" {
		patternFile, err := filepath.Abs(*c.PatternFile)
		if err != nil {
			return fmt.Errorf("pattern file path: %w", err)
		}
		*c.PatternFile = patternFile

		patterns, err := loadPatternsFromFile(*c.PatternFile)
		if err != nil {
			return fmt.Errorf("loading patterns from file: %w", err)
		}
		c.AllowedPattern.Merge(patterns)
	}
	return nil
}

func loadPatternsFromFile(path string) (patterns patternSet, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening pattern file: %w", err)
	}
	defer func() {
		if e := file.Close(); e != nil && err == nil {
			err = e
		}
	}()

	patterns = make(patternSet)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		re := regexp.MustCompile(scanner.Text())
		patterns[re] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading pattern file: %w", err)
	}
	return patterns, nil
}

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

	config.PatternFile = flag.String(
		"pattern-file",
		"",
		"bypass DPI only on packets matching regex patterns provided in a file (one per line)",
	)

	allowedPatterns := make(StringSet)
	flag.Var(
		&allowedPatterns,
		"pattern",
		"bypass DPI only on packets matching this regex pattern; can be given multiple times",
	)
	flag.Parse()

	if len(allowedPatterns) > 0 {
		config.AllowedPattern = make(patternSet, len(allowedPatterns))
	}
	for pattern := range allowedPatterns {
		config.AllowedPattern[regexp.MustCompile(pattern)] = struct{}{}
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
