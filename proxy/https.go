package proxy

import (
	"fmt"
	"log"
	"net"
    "io"

	// "time"

	"github.com/xvzc/SpoofDPI/util"
)

func HandleHttps(clientConn net.Conn, ip string) {
    remoteConn, err := net.Dial("tcp", ip+":443") // create connection to server
    if err != nil {
        log.Fatal(err)
        return
    }
    defer remoteConn.Close()

    log.Println("Connected to the server.")

    // established := []byte("HTTP/1.1 204 No Content\n\n")

    log.Println("Sending 200 Connection Estabalished")

    fmt.Fprintf(clientConn, "HTTP/1.1 200 Connection Established\r\n\r\n")


    go func() {
        for {
            buf, err := util.ReadMessage(remoteConn)
            if err != nil {
                if err != io.EOF {
                    log.Println("Error reading from the server:", err)
                } else {
                    log.Println("Remote connection Closed: ", err)
                }
                return
            }

            log.Println("Server Sent Data", len(buf))

            _, write_err := clientConn.Write(buf)
            if write_err != nil {
                log.Println("Error writing to client:", write_err)
                return
            }
        }
    }()

    for {
        defer clientConn.Close()
        buf, err := util.ReadMessage(clientConn)
        if err != nil {
            if err != io.EOF {
                log.Println("Error reading from the client:", err)
            } else {
                log.Println("Client connection Closed: ", err)
            }
            break
        }
        log.Println("Client Sent Data", len(buf))

        _, write_err := remoteConn.Write(buf)
        if write_err != nil {
            log.Println("Error writing to client:", write_err)
            break
        }
    }

}
