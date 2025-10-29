package handler

import (
	"context"
	"github.com/xvzc/SpoofDPI/packet"
	"net"
)

type Handler interface {
	Serve(ctx context.Context, lConn *net.TCPConn, initPkt *packet.HttpRequest, ip string)
	Protocol() string
}
