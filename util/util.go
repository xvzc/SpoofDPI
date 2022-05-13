package util

import (
	"flag"
	"fmt"

	"github.com/pterm/pterm"
)

func ParseArgs() (string, int, string, bool) {
	addr := flag.String("addr", "127.0.0.1", "Listen addr")
	port := flag.Int("port", 8080, "port")
	dns := flag.String("dns", "8.8.8.8", "DNS server")
	debug := flag.Bool("debug", false, "true | false")

	flag.Parse()

	return *addr, *port, *dns, *debug
}

func PrintWelcome(addr string, port int, dns string, debug bool) {
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
