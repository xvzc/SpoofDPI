package proxy

import (
	"log"
	"net"
	"os"

	"github.com/xvzc/SpoofDPI/util"
	"github.com/xvzc/SpoofDPI/config"
)

func Start() {
    listener, err := net.Listen("tcp", ":" + config.GetConfig().SrcPort)
	if err != nil {
        log.Fatal("Error creating listener: ", err)
        os.Exit(1)
	}

    log.Println("Created a listener")

    for {
		clientConn, err := listener.Accept()
		if err != nil {
            log.Fatal("Error accepting connection: ", err)
			continue
		}

        log.Println("Accepted a new connection.", clientConn.RemoteAddr())

        go func() {
            defer clientConn.Close()

            message , err := util.ReadMessage(clientConn)
            if err != nil {
                return
            }

            log.Println("Client sent data: ", len(message))

            domain := util.ExtractDomain(&message)

            ip, err := util.DnsLookupOverHttps(config.GetConfig().DNS, domain) // Dns lookup over https
            if err != nil {
                return
            }

            log.Println("ip: "+ ip)

            if util.ExtractMethod(&message) == "CONNECT" {
                util.Debug("HTTPS Requested")
                HandleHttps(clientConn, ip)
            }else {
                log.Println("HTTP Requested.")
                HandleHttp(clientConn, ip, message)
            }
        }()
    }
}
