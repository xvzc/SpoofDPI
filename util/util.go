package util

import (
	"flag"
	"fmt"

	"github.com/pterm/pterm"
)

func ParseArgs() (string, string, bool) {
	port := flag.String("port", "8080", "port")
	dns := flag.String("dns", "8.8.8.8", "DNS server")
	debug := flag.Bool("debug", false, "true | false")

	flag.Parse()

	return *port, *dns, *debug
}

func BytesToChunks(buf []byte) [][]byte {
	if len(buf) < 1 {
		return [][]byte{buf}
	}

	return [][]byte{buf[:1], buf[1:]}
}

func PrintWelcome(port string, dns string, debug bool) {
	cyan := pterm.NewLettersFromStringWithStyle("Spoof", pterm.NewStyle(pterm.FgCyan))
	purple := pterm.NewLettersFromStringWithStyle("DPI", pterm.NewStyle(pterm.FgLightMagenta))
	pterm.DefaultBigText.WithLetters(cyan, purple).Render()

	pterm.DefaultBulletList.WithItems([]pterm.BulletListItem{
		{Level: 0, Text: "PORT  : " + port},
		{Level: 0, Text: "DNS   : " + dns},
		{Level: 0, Text: "DEBUG : " + fmt.Sprint(debug)},
	}).Render()
}
