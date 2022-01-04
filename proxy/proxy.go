package proxy

import (
	"log"
	"net"
	"os"

	"github.com/xvzc/SpoofDPI/config"
	"github.com/xvzc/SpoofDPI/util"
)

func Start() {
	listener, err := net.Listen("tcp", ":"+config.GetConfig().Port)
	if err != nil {
		log.Fatal("Error creating listener: ", err)
		os.Exit(1)
	}

	util.Debug("Created a listener")

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			log.Fatal("Error accepting connection: ", err)
			continue
		}

		util.Debug("Accepted a new connection.", clientConn.RemoteAddr())

		go func() {
			defer clientConn.Close()

			message, err := ReadBytes(clientConn)
			if err != nil {
				return
			}

			util.Debug("Client sent data: ", len(message))

			domain := util.ExtractDomain(&message)

			ip, err := util.DnsLookupOverHttps(config.GetConfig().DNS, domain) // Dns lookup over https
			if err != nil {
				return
			}

			util.Debug("ip: " + ip)

			if util.ExtractMethod(&message) == "CONNECT" {
				util.Debug("HTTPS Requested")
				HandleHttps(clientConn, ip)
			} else {
				util.Debug("HTTP Requested.")
				HandleHttp(clientConn, ip, message)
			}
		}()
	}
}
