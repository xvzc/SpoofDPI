package handler

import (
	"fmt"
    "io/ioutil"
	"log"
	"net"
    "SpoofDPI/util"


	"github.com/babolivier/go-doh-client"
)

var resolver = doh.Resolver{
    Host:  "8.8.8.8",
    Class: doh.IN,
}

func HandleClientRequest(clientConn net.Conn) {
    defer clientConn.Close()

    buf, err := util.ReadBytes(clientConn)
    if err != nil {
        return
    }

    fmt.Println("\n##### Request from client : ")
    fmt.Println(string(buf))

    domain, port := util.ExtractDomainAndPort(string(buf))

    log.Println("domain: "+ domain)
    log.Println("port: " + port)

    ip, err := util.DnsLookupOverHttps(domain) // Dns lookup over https
    if err != nil {
        log.Fatal(err)
        return
    }

    remoteConn, err := net.Dial("tcp", ip+":"+port) // create connection to server
    if err != nil {
        fmt.Println(err)
        return
    }
    defer remoteConn.Close()

    DoMitm(clientConn, remoteConn, buf)
}

func DoMitm(clientConn net.Conn, remoteConn net.Conn, data []byte) {
    _, write_err := remoteConn.Write(data)
    if write_err != nil {
        fmt.Println("failed:", write_err)
        return
    }
    defer remoteConn.(*net.TCPConn).CloseWrite()

    buf, read_err := ioutil.ReadAll(remoteConn)
    if read_err != nil {
        fmt.Println("failed:", read_err)
        return
    }

    log.Println("\n##### Response from server: ")
    log.Println(string(buf))

    _, write_err = clientConn.Write(buf)
    if write_err != nil {
        fmt.Println("failed:", write_err)
        return
    }
    defer clientConn.(*net.TCPConn).CloseWrite()
}
