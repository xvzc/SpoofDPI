package util

import (
	"flag"
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
