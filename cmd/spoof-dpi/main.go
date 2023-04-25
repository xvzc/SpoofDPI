package main

import (
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/xvzc/SpoofDPI/doh"
	"github.com/xvzc/SpoofDPI/packet"
	"github.com/xvzc/SpoofDPI/proxy"
	"github.com/xvzc/SpoofDPI/util"
)

func main() {
	addr, port, dns, debug, banner, allowedHosts, allowedPattern := util.ParseArgs()

	if(len(*allowedHosts) > 0) {
		var escapedUrls []string
		for _, host := range *allowedHosts {
			escapedUrls = append(escapedUrls, regexp.QuoteMeta(host))
		}

		allowedHostsRegex := strings.Join(escapedUrls, "|")
		packet.UrlsMatcher = regexp.MustCompile(allowedHostsRegex)
	}

	if(allowedPattern != "") {
		packet.PatternMatcher = regexp.MustCompile(allowedPattern)
	}

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
