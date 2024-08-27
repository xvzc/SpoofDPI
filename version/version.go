package version

import _ "embed"

//go:embed VERSION
var VERSION string

func PrintVersion() {
	println("spoofdpi", "v" + VERSION)
	println("A simple and fast anti-censorship tool written in Go.")
	println("https://github.com/xvzc/SpoofDPI")
}
