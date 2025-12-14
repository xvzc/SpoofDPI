package proxy

import (
	"context"
	"net"
	"time"

	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/proto"
)

type Handler interface {
	HandleRequest(
		ctx context.Context,
		lConn net.Conn,
		req *proto.HTTPRequest,
		dst *Destination,
		rule *config.Rule,
	) error
}

type Destination struct {
	Domain  string
	Addrs   []net.IPAddr
	Port    int
	Timeout time.Duration
}
