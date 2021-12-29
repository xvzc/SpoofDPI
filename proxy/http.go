package proxy

import (
	"fmt"
	"log"
	"net"

	"github.com/xvzc/SpoofDPI/util"
)

func HandleHttp(clientConn net.Conn, ip string, message []byte)  {
    remoteConn, err := net.Dial("tcp", ip+":80") // create connection to server
    if err != nil {
        log.Fatal(err)
        return
    }
    defer remoteConn.Close()

    _, write_err := remoteConn.Write(message)
    if write_err != nil {
        log.Fatal("failed:", write_err)
        return
    }
    defer remoteConn.(*net.TCPConn).CloseWrite()

    buf, err := util.ReadMessage(remoteConn)
    if err != nil {
        log.Fatal("failed:", err)
        return
    }

    fmt.Println()
    log.Println()
    fmt.Println("##### Response from the server: ")
    fmt.Println(string(buf))

    // Write to client
    _, write_err = clientConn.Write(buf)
    if write_err != nil {
        log.Fatal("failed:", write_err)
        return
    }
    defer clientConn.(*net.TCPConn).CloseWrite()
}

