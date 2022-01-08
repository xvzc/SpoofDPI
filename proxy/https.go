package proxy

import (
	"fmt"
	"net"

	"github.com/xvzc/SpoofDPI/packet"
	"github.com/xvzc/SpoofDPI/util"
)

func HandleHttps(clientConn net.Conn, ip string, r *packet.HttpRequest) {
	// Create a connection to the requested server
	remoteConn, err := net.Dial("tcp", ip+":443")
	if err != nil {
		util.Debug(err)
		return
	}
	defer remoteConn.Close()

	util.Debug("[HTTPS] Connected to the server.")

	// Send self generated response for connect request
	fmt.Fprintf(clientConn, "HTTP/1.1 200 Connection Established\r\n\r\n")
	util.Debug("[HTTPS] Sent 200 Connection Estabalished")

	// Read client hello
	clientHello, err := ReadBytes(clientConn)
	if err != nil {
		util.Debug("[HTTPS] Error reading client hello: ", err)
		util.Debug("Closing connection ", clientConn.RemoteAddr())
	}

	util.Debug(clientConn.RemoteAddr(), "[HTTPS] Client sent hello", len(clientHello))

	// Generate a go routine that reads from the server
	go Serve(remoteConn, clientConn, "HTTPS")

	// Send chunked request
	chunks := util.BytesToChunks(clientHello)
	for i := 0; i < len(chunks); i++ {
		_, write_err := remoteConn.Write(chunks[i])
		if write_err != nil {
			util.Debug("[HTTPS] Error writing to the client:", write_err)
			break
		}
	}

	// Read from the client
	Serve(clientConn, remoteConn, "HTTPS")
}
