package proxy

import (
	"fmt"
	"net"
    "io"

	// "time"

	"github.com/xvzc/SpoofDPI/util"
)

func HandleHttps(clientConn net.Conn, ip string) {
    remoteConn, err := net.Dial("tcp", ip+":443") // create connection to server
    if err != nil {
        util.Debug(err)
        return
    }
    defer remoteConn.Close()

    util.Debug("Connected to the server.")

    util.Debug("Sending 200 Connection Estabalished")

    fmt.Fprintf(clientConn, "HTTP/1.1 200 Connection Established\r\n\r\n")


    go func() {
        for {
            buf, err := util.ReadMessage(remoteConn)
            if err != nil {
                if err != io.EOF {
                    util.Debug("Error reading from the server:", err)
                } else {
                    util.Debug("Remote connection Closed: ", err)
                }

                util.Debug("Closing connection: ", remoteConn.RemoteAddr())
                return
            }

            util.Debug(remoteConn.RemoteAddr(), "Server Sent Data", len(buf))

            _, write_err := clientConn.Write(buf)
            if write_err != nil {
                util.Debug("Error writing to client:", write_err)
                return
            }
        }
    }()

    for {
        defer clientConn.Close()
        buf, err := util.ReadMessage(clientConn)
        if err != nil {
            if err != io.EOF {
                util.Debug("Error reading from the client:", err)
            } else {
                util.Debug("Client connection Closed: ", err)
            }

            util.Debug("Closing connection: ", clientConn.RemoteAddr())
            break
        }
        util.Debug(clientConn.RemoteAddr(), "Client Sent Data", len(buf))

        _, write_err := remoteConn.Write(buf)
        if write_err != nil {
            util.Debug("Error writing to client:", write_err)
            break
        }
    }
}
