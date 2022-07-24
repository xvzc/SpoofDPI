package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/xvzc/SpoofDPI/doh"
	"github.com/xvzc/SpoofDPI/proxy"
	"github.com/xvzc/SpoofDPI/util"
)

func main() {
	addr, port, dns, debug, banner := util.ParseArgs()

	p := proxy.New(addr, port)
	doh.Init(dns)
	if debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

    if banner {
        util.PrintColoredBanner(addr, port, dns, debug)
    } else {
        util.PrintSimpleInfo(addr, port, dns, debug)
    }

	if err := util.SetOsProxy(port); err != nil {
		log.Fatal(err)
	}

	go p.Start()

	// Handle signals
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(
		sigs,
		syscall.SIGKILL,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP)

	go func() {
		_ = <-sigs
		done <- true
	}()

	<-done
	if err := util.UnsetOsProxy(); err != nil {
		log.Fatal(err)
	}
}
