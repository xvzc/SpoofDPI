package proxy

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/xvzc/SpoofDPI/net"
	"github.com/xvzc/SpoofDPI/packet"
)

func HandleHttp(clientConn net.Conn, ip string, p *packet.HttpPacket) {
	// Create connection to server
	remoteConn, err := net.Dial("tcp", ip+":80")
	if err != nil {
		log.Debug(err)
		return
	}
	defer remoteConn.Close()

	log.Debug("[HTTP] Connected to the server.")

	go remoteConn.Serve(clientConn, "HTTP")

	log.Debug("[HTTP] Sending request to the server")
	fmt.Fprintf(remoteConn.Conn, string(p.Raw))

	go clientConn.Serve(remoteConn, "HTTP")
}

func HandleHttps(clientConn net.Conn, ip string, r *packet.HttpPacket) {
	// Create a connection to the requested server
	remoteConn, err := net.Dial("tcp", ip+":443")
	if err != nil {
		log.Debug(err)
		return
	}
	defer remoteConn.Close()

	log.Debug("[HTTPS] Connected to the server.")

	// Send self generated response for connect request
	fmt.Fprintf(clientConn.Conn, "HTTP/1.1 200 Connection Established\r\n\r\n")
	log.Debug("[HTTPS] Sent 200 Connection Estabalished")

	// Read client hello
	clientHello, err := clientConn.ReadBytes()
	if err != nil {
		log.Debug("[HTTPS] Error reading client hello: ", err)
		log.Debug("Closing connection: ", clientConn.RemoteAddr())
	}

	log.Debug(clientConn.RemoteAddr(), "[HTTPS] Client sent hello: ", len(clientHello), "bytes")

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
