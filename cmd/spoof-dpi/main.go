package main

import (
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/xvzc/SpoofDPI/proxy"
	"github.com/xvzc/SpoofDPI/util"
)

func main() {
	port, dns, debug := util.ParseArgs()

	p := proxy.New(port, dns, runtime.GOOS, debug)
	p.PrintWelcome()

	if err := p.SetOsProxy(); err != nil {
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
	if err := p.UnsetOsProxy(); err != nil {
		log.Fatal(err)
	}
}
