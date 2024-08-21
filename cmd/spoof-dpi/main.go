package main

import (
	"context"
	"github.com/xvzc/SpoofDPI/util/log"
	"os"
	"os/signal"
	"syscall"

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
	config.Load(args)

	log.InitLogger(config)

	pxy := proxy.New(config)

	if *config.NoBanner {
		util.PrintSimpleInfo()
	} else {
		util.PrintColoredBanner()
	}

	if *config.SystemProxy {
		if err := util.SetOsProxy(*config.Port); err != nil {
			log.Logger.Fatal().Msgf("error while changing proxy settings: %s", err)
		}
	}

	go pxy.Start(context.Background())

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
			log.Logger.Fatal().Msgf("%s", err)
		}
	}
}
