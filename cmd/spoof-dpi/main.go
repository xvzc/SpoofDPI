package main

import (
	"flag"
	"os"

	"github.com/pterm/pterm"
	"github.com/xvzc/SpoofDPI/config"
	"github.com/xvzc/SpoofDPI/proxy"
)

func main() {
    src := flag.String("src", "127.0.0.1:8080", "source-ip:source-port")
    dns := flag.String("dns", "8.8.8.8", "DNS server")
    debug := flag.Bool("debug", false, "true | false")

    flag.Parse()

    err := config.InitConfig(*src, *dns, *debug)
    if err != nil {
        os.Exit(1)
    }

    cyan := pterm.NewLettersFromStringWithStyle("Spoof", pterm.NewStyle(pterm.FgCyan))
    purple := pterm.NewLettersFromStringWithStyle("DPI", pterm.NewStyle(pterm.FgLightMagenta))
    pterm.DefaultBigText.WithLetters(cyan, purple).Render()

    pterm.DefaultBulletList.WithItems([]pterm.BulletListItem{
        {Level: 0, Text: "SRC : " + *src},
        {Level: 0, Text: "DNS : " + *dns},
    }).Render()

    proxy.Start()
}
