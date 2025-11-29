package desync

import (
	"context"
	"net"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/proto"
)

// TLSDefault writes the data to the connection without modification.
type TLSDefault struct{}

func NewTLSDefault() TLSDesyncer {
	return &TLSDefault{}
}

func (s *TLSDefault) Send(
	ctx context.Context,
	logger zerolog.Logger,
	conn net.Conn,
	msg *proto.TLSMessage,
) (int, error) {
	return conn.Write(msg.Raw)
}

func (s *TLSDefault) String() string {
	return "default"
}
