package proxy

import (
	"fmt"
	"log"
	"net"

	// "time"

	"github.com/xvzc/SpoofDPI/util"
)

func HandleHttps(clientConn net.Conn, ip string) {

    remoteConn, err := net.Dial("tcp", ip+":443") // create connection to server
    if err != nil {
        log.Fatal(err)
        return
    }
    log.Println("Connected to the server.")

    // established := []byte("HTTP/1.1 204 No Content\n\n")

    log.Println("Sending 204 No Content to the client..")

    fmt.Fprintf(clientConn, "HTTP/1.1 204 No Content\r\n\r\n")


    go func() {
        defer remoteConn.Close()
        for {
            buf, err := util.ReadMessage(remoteConn)
            if err != nil {
                log.Println(err)
                return
            }

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
            log.Println(err)
            break
        }

        _, write_err := remoteConn.Write(buf)
        if write_err != nil {
            log.Println("Error writing to client:", write_err)
            break
        }
    }

    /*
    serverHello, err := util.WriteAndRead(remoteConn, clientHello)
    log.Println("Server sent data. length:", len(serverHello))

    clientFinish, err := util.WriteAndRead(clientConn, serverHello)
    log.Println("Client sent data. length:", len(clientFinish))

    _, err = remoteConn.Write(clientFinish)
    if err != nil {
        log.Fatal("Error writing to client:", err)
    }

    log.Println("Written")
    */

}
