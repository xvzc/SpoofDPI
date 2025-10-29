package util

import (
	"fmt"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
	"github.com/xvzc/SpoofDPI/config"
)

func PrintColoredBanner() {
	config := config.Get()
	cyan := putils.LettersFromStringWithStyle("Spoof", pterm.NewStyle(pterm.FgCyan))
	purple := putils.LettersFromStringWithStyle("DPI", pterm.NewStyle(pterm.FgLightMagenta))
	pterm.DefaultBigText.WithLetters(cyan, purple).Render()

	_ = pterm.DefaultBulletList.WithItems([]pterm.BulletListItem{
		{Level: 0, Text: "ADDR    : " + fmt.Sprint(config.Addr())},
		{Level: 0, Text: "PORT    : " + fmt.Sprint(config.Port())},
		{Level: 0, Text: "DNS     : " + fmt.Sprint(config.DnsAddr())},
		{Level: 0, Text: "DEBUG   : " + fmt.Sprint(config.Debug())},
	}).Render()

	pterm.DefaultBasicText.Println("Press 'CTRL + c' to quit")
}
