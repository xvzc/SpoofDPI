package version

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var VERSION string

func Version() string {
	return "v" + strings.Trim(VERSION, "\n\t ")
}

func PrintVersion() {
	println("spoofdpi- Simple and fast anti-censorship tool written in Go.")
	println(Version())
	println("https://github.com/xvzc/SpoofDPI")
}
