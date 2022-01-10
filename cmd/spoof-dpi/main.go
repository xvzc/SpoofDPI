package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/xvzc/SpoofDPI/doh"
	"github.com/xvzc/SpoofDPI/proxy"
	"github.com/xvzc/SpoofDPI/util"
)

func main() {
	port, dns, debug := util.ParseArgs()

	p := proxy.New(port)
	util.PrintWelcome(port, dns, debug)

	if err := util.SetOsProxy(port); err != nil {
		log.Fatal(err)
	}

	doh.Init(dns)

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
