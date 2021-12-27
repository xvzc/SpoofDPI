package main

import (
	"SpoofDPI/mitm"
	"SpoofDPI/util"
    "fmt"
	"log"
	"net"
)

const (
    CLI_PORT = "8080"
    DNS_ADDR = "1.1.1.1"
)

func main() {
	log.Println("##### Listening 8080..")

    listener, err := net.Listen("tcp", ":" + CLI_PORT)
	if err != nil {
		panic(err)
	}

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			log.Println("error accepting connection", err)
			continue
		}

		log.Println("##### New connection", clientConn.RemoteAddr())

        go func() {
            defer clientConn.Close()

            buf, err := util.ReadBytes(clientConn)
            if err != nil {
                return
            }

            fmt.Println()
            log.Println()
            fmt.Println("##### Request from client : ")
            fmt.Println(string(buf))

            domain, port := util.ExtractDomainAndPort(string(buf))

            log.Println("domain: "+ domain)
            log.Println("port: " + port)

            ip, err := util.DnsLookupOverHttps(DNS_ADDR, domain) // Dns lookup over https
            if err != nil {
                log.Fatal(err)
                return
            }

            remoteConn, err := net.Dial("tcp", ip+":"+port) // create connection to server
            if err != nil {
                log.Fatal(err)
                return
            }
            defer remoteConn.Close()

            mitm.GoGoSing(clientConn, remoteConn, buf)
        }()
	}
}
