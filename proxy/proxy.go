package proxy

import (
	"log"
	"net"
	"os"

	"github.com/xvzc/SpoofDPI/config"
	"github.com/xvzc/SpoofDPI/packet"
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

			b, err := ReadBytes(clientConn)
			if err != nil {
				return
			}

			util.Debug("Client sent data: ", len(b))

			r := packet.NewHttpRequest(&b)
			util.Debug("Request: \n" + string(*r.Raw))

			if !r.IsValidMethod() {
				log.Println("Unsupported method: ", r.Method)
				return
			}

			// Dns lookup over https
			ip, err := util.DnsLookupOverHttps(config.GetConfig().DNS, r.Domain)
			if err != nil {
				log.Println("Error looking up dns: "+r.Domain, err)
				return
			}

			util.Debug("ip: " + ip)

			if r.IsConnectMethod() {
				util.Debug("HTTPS Requested")
				HandleHttps(clientConn, ip, &r)
			} else {
				util.Debug("HTTP Requested.")
				HandleHttp(clientConn, ip, &r)
			}
		}()
	}
}
