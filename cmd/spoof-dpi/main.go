package main

import (
	"fmt"
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
	fmt.Println(*p)

	p.PrintWelcome()

	err := p.SetOsProxy()
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	go p.Start()

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
	err = p.UnsetOsProxy()
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
