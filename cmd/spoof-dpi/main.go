package main

import (
	"flag"
	"os"

	"github.com/xvzc/SpoofDPI/proxy"
	"github.com/xvzc/SpoofDPI/config"
)

func main() {
    src := flag.String("src", "localhost:8080", "source-ip:source-port")
    dns := flag.String("dns", "8.8.8.8", "DNS server")
    debug := flag.Bool("debug", false, "true | false")
    mtu := flag.Int("mtu", 100, "int")

    err := config.InitConfig(*src, *dns, *mtu, *debug)
    if err != nil {
        os.Exit(1)
    }

    proxy.Start()
}
