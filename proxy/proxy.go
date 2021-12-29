package proxy

import (
	"log"
    "fmt"
	"net"
	"os"
    "github.com/xvzc/SpoofDPI/util"
)

func Start() {
    listener, err := net.Listen("tcp", ":" + config.SrcPort)
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

            fmt.Println()
            log.Println()
            fmt.Println("##### Request from client : ")
            fmt.Println(string(message))

            domain := util.ExtractDomain(&message)

            ip, err := util.DnsLookupOverHttps(getConfig().DNS, domain) // Dns lookup over https
            if err != nil {
                return
            }

            log.Println("ip: "+ ip)

            if util.ExtractMethod(&message) == "CONNECT" {
                fmt.Println("got a HTTPS Request")
            }else {
                HandleHttp(clientConn, ip, message)
            }
        }()
    }
}
