package main

import (
	"os"
	"os/signal"
	"syscall"
  _ "embed"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/xvzc/SpoofDPI/proxy"
	"github.com/xvzc/SpoofDPI/util"
	"github.com/xvzc/SpoofDPI/version"
)

func main() {
	util.ParseArgs()
	config := util.GetConfig()
	if *config.Version {
    PrintVersion()
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
