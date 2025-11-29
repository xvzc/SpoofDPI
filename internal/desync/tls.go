package desync

import (
	"context"
	"net"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/proto"
)

// TLSDesyncer defines the interface for manipulating and sending packets.
// Implementations can be composed using the Decorator pattern.
type TLSDesyncer interface {
	Send(
		ctx context.Context,
		logger zerolog.Logger,
		conn net.Conn,
		msg *proto.TLSMessage,
	) (int, error)
	String() string
}
