package proxy

import (
	"context"
	"net"
	"time"

	"github.com/xvzc/SpoofDPI/internal/proto"
)

type Handler interface {
	HandleRequest(
		ctx context.Context,
		lConn net.Conn,
		req *proto.HTTPRequest,
		dstAddrs []net.IPAddr,
		dstPort int,
		timeout time.Duration,
	) error
}
