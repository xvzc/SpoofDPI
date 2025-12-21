package socks5

import (
	"context"
	"net"

	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/proto"
)

type Handler interface {
	Handle(ctx context.Context, conn net.Conn, req *proto.SOCKS5Request, rule *config.Rule, addrs []net.IPAddr) error
}