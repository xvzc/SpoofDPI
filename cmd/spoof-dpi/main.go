package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/xvzc/SpoofDPI/proxy"
	"github.com/xvzc/SpoofDPI/util"
	"github.com/xvzc/SpoofDPI/version"
)

func main() {
	args := util.ParseArgs()
	if *args.Version {
		version.PrintVersion()
		os.Exit(0)
	}

	config := util.GetConfig()
	if err := config.Load(args); err != nil {
		log.Fatalf("loading config: %s", err)
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
			log.Fatal("error while changing proxy settings")
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
