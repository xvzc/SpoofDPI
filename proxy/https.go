package proxy

import (
	"fmt"

	"github.com/xvzc/SpoofDPI/net"
	"github.com/xvzc/SpoofDPI/packet"
)

func HandleHttp(clientConn net.Conn, ip string, p *packet.HttpPacket) {
	remoteConn, err := net.Dial("tcp", ip+":80") // create connection to server
	if err != nil {
		// util.Debug(err)
		return
	}
	defer remoteConn.Close()

	// util.Debug("[HTTP] Connected to the server.")

	go remoteConn.Serve(clientConn, "HTTP")

	// util.Debug("[HTTP] Sending request to the server")
	fmt.Fprintf(remoteConn.Conn, string(p.Raw))

	go clientConn.Serve(remoteConn, "HTTP")
}

func HandleHttps(clientConn net.Conn, ip string, r *packet.HttpPacket) {
	// Create a connection to the requested server
	remoteConn, err := net.Dial("tcp", ip+":443")
	if err != nil {
		// util.Debug(err)
		return
	}
	defer remoteConn.Close()

	// util.Debug("[HTTPS] Connected to the server.")

	// Send self generated response for connect request
	fmt.Fprintf(clientConn.Conn, "HTTP/1.1 200 Connection Established\r\n\r\n")
	// util.Debug("[HTTPS] Sent 200 Connection Estabalished")

	// Read client hello
	clientHello, err := clientConn.ReadBytes()
	if err != nil {
		// util.Debug("[HTTPS] Error reading client hello: ", err)
		// util.Debug("Closing connection ", clientConn.RemoteAddr())
	}

	// util.Debug(clientConn.RemoteAddr(), "[HTTPS] Client sent hello", len(clientHello))

	// Generate a go routine that reads from the server
	go remoteConn.Serve(clientConn, "HTTPS")

	pkt := packet.NewHttpsPacket(clientHello)

	chunks := pkt.SplitInChunks()

	if _, err := remoteConn.WriteChunks(chunks); err != nil {
		return
	}

	// Read from the client
	clientConn.Serve(remoteConn, "HTTPS")
}
