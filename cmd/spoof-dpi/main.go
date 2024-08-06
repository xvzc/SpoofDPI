package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/xvzc/SpoofDPI/proxy"
	"github.com/xvzc/SpoofDPI/util"
)

var VERSION = "v0.0.0(dev)"
func main() {
	util.ParseArgs()
	config := util.GetConfig()
	if *config.Version {
		println("spoof-dpi", VERSION)
		println("\nA simple and fast anti-censorship tool written in Go.")
		println("https://github.com/xvzc/SpoofDPI")
		os.Exit(0)
	}

	pxy := proxy.New(config)
	if *config.Debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	if *config.NoBanner {
		util.PrintSimpleInfo()
	} else {
		util.PrintColoredBanner()
	}

	if *config.SystemProxy {
		if err := util.SetOsProxy(*config.Port); err != nil {
			log.Fatal("Error while changing proxy settings")
		}
	}

	go pxy.Start()

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

	if *config.SystemProxy {
		if err := util.UnsetOsProxy(); err != nil {
			log.Fatal(err)
		}
	}
}
