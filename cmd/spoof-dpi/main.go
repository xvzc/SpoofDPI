package main

import (
	"flag"
	"log"
	"os"

	"github.com/xvzc/SpoofDPI/config"
	"github.com/xvzc/SpoofDPI/proxy"
    "github.com/xvzc/SpoofDPI/util"
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

    util.PrintWelcome()

    err = config.SetOsProxy()
    if err != nil {
        log.Fatal(err)
        os.Exit(1)
    }

    proxy.Start()
}
