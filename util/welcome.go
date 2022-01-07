package util

import (
	"github.com/pterm/pterm"
	"github.com/xvzc/SpoofDPI/config"
)

func PrintWelcome() {
	cyan := pterm.NewLettersFromStringWithStyle("Spoof", pterm.NewStyle(pterm.FgCyan))
	purple := pterm.NewLettersFromStringWithStyle("DPI", pterm.NewStyle(pterm.FgLightMagenta))
	pterm.DefaultBigText.WithLetters(cyan, purple).Render()

	pterm.DefaultBulletList.WithItems([]pterm.BulletListItem{
		{Level: 0, Text: "PORT : " + config.GetConfig().Port},
		{Level: 0, Text: "DNS  : " + config.GetConfig().DNS},
	}).Render()

}
