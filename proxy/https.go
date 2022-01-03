package proxy

import (
	"fmt"
	"io"
	"net"

	// "time"

	"github.com/xvzc/SpoofDPI/config"
	"github.com/xvzc/SpoofDPI/util"
)

func HandleHttps(clientConn net.Conn, ip string) {
    remoteConn, err := net.Dial("tcp", ip+":443") // create connection to server
    if err != nil {
        util.Debug(err)
        return
    }
    defer clientConn.Close()
    defer remoteConn.Close()

    util.Debug("Connected to the server.")

    go func() {
        for {
            buf, err := util.ReadMessage(remoteConn)
            if err != nil {
                util.Debug("Error reading from the server", err, " Closing connection ", remoteConn.RemoteAddr())
                return
            }

            util.Debug(remoteConn.RemoteAddr(), "Server sent data", len(buf))

            _, write_err := clientConn.Write(buf)
            if write_err != nil {
                util.Debug("Error sending data to the client:", write_err)
                return
            }
        }
    }()

    util.Debug("Sending 200 Connection Estabalished")
    fmt.Fprintf(clientConn, "HTTP/1.1 200 Connection Established\r\n\r\n")

    clientHello, err := util.ReadMessage(clientConn)
    if err != nil {
        util.Debug("Error reading client hello", err, " Closing connection ", clientConn.RemoteAddr())
    }

    util.Debug(clientConn.RemoteAddr(), "Client sent hello", len(clientHello))

    chunks, err := util.SplitInChunks(clientHello, config.GetConfig().MTU)
    if err != nil {
        util.Debug("Error chunking client hello: ", err)
    }

    for i := 0; i < len(chunks); i++ {
        _, write_err := remoteConn.Write(chunks[i])
        if write_err != nil {
            util.Debug("Error writing to client:", write_err)
            break
        }
    }

    for {
        buf, err := util.ReadMessage(clientConn)
        if err != nil {
            util.Debug("Error reading from the client", err, " Closing connection ", clientConn.RemoteAddr())
            break
        }

        util.Debug(clientConn.RemoteAddr(), "Client sent data", len(buf))

        _, write_err := remoteConn.Write(buf)
        if write_err != nil {
            util.Debug("Error writing to client:", write_err)
            break
        }
    }
}
