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
    port := flag.String("port", "8080", "port")
    dns := flag.String("dns", "8.8.8.8", "DNS server")
    debug := flag.Bool("debug", false, "true | false")

    flag.Parse()

    err := config.InitConfig(*port, *dns, *debug)
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
