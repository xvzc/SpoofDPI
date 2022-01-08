package proxy

import (
	"fmt"
	"net"

	"github.com/xvzc/SpoofDPI/packet"
	"github.com/xvzc/SpoofDPI/util"
)

func HandleHttp(clientConn net.Conn, ip string, p *packet.HttpRequest) {
	remoteConn, err := net.Dial("tcp", ip+":80") // create connection to server
	if err != nil {
		util.Debug(err)
		return
	}
	defer remoteConn.Close()

	util.Debug("[HTTP] Connected to the server.")

	go Serve(remoteConn, clientConn, "HTTP")

	util.Debug("[HTTP] Sending request to the server")
	fmt.Fprintf(remoteConn, string(*p.Raw))

	Serve(clientConn, remoteConn, "HTTP")
}
