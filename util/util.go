package util

import (
	"flag"
	"fmt"

	"github.com/pterm/pterm"
)

type ArrayFlags []string

func (i *ArrayFlags) String() string {
    return "my string representation"
}

func (i *ArrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func ParseArgs() (string, int, string, bool, bool, *ArrayFlags, string) {
	addr := flag.String("addr", "127.0.0.1", "Listen addr")
	port := flag.Int("port", 8080, "port")
	dns := flag.String("dns", "8.8.8.8", "DNS server")
	debug := flag.Bool("debug", false, "true | false")
	banner := flag.Bool("banner", true, "true | false")

	var allowedUrls ArrayFlags
	flag.Var(&allowedUrls, "url", "Bypass DPI only on this url, can be passed multiple times")
	allowedPattern := flag.String("pattern", "", "Bypass DPI only on packets matching this regex pattern")

	flag.Parse()

	return *addr, *port, *dns, *debug, *banner, &allowedUrls, *allowedPattern
}

func PrintColoredBanner(addr string, port int, dns string, debug bool) {
	cyan := pterm.NewLettersFromStringWithStyle("Spoof", pterm.NewStyle(pterm.FgCyan))
	purple := pterm.NewLettersFromStringWithStyle("DPI", pterm.NewStyle(pterm.FgLightMagenta))
	pterm.DefaultBigText.WithLetters(cyan, purple).Render()

	pterm.DefaultBulletList.WithItems([]pterm.BulletListItem{
		{Level: 0, Text: "ADDR  : " + addr},
		{Level: 0, Text: "PORT  : " + fmt.Sprint(port)},
		{Level: 0, Text: "DNS   : " + dns},
		{Level: 0, Text: "DEBUG : " + fmt.Sprint(debug)},
	}).Render()
}

func PrintSimpleInfo(addr string, port int, dns string, debug bool) {
	fmt.Println("")
	fmt.Println("- ADDR  : ", addr)
	fmt.Println("- PORT  : ", port)
	fmt.Println("- DNS   : ", dns)
	fmt.Println("- DEBUG : ", debug)
	fmt.Println("")
}
