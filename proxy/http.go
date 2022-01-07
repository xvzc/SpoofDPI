package proxy

import (
	"fmt"
	"net"

	"github.com/xvzc/SpoofDPI/request"
	"github.com/xvzc/SpoofDPI/util"
)

func HandleHttp(clientConn net.Conn, ip string, r *request.HttpRequest) {
	remoteConn, err := net.Dial("tcp", ip+":80") // create connection to server
	if err != nil {
		util.Debug(err)
		return
	}
	defer remoteConn.Close()

	util.Debug("[HTTP] Connected to the server.")

	go Serve(remoteConn, clientConn, "HTTP")

	util.Debug("[HTTP] Sending request to the server")
	fmt.Fprintf(remoteConn, string(*r.Raw))

	Serve(clientConn, remoteConn, "HTTP")
}
