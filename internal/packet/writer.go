package packet

import (
	"context"
	"net"
)

type Writer interface {
	WriteCraftedPacket(
		ctx context.Context,
		src net.Addr,
		dst net.Addr,
		ttl uint8,
		payload []byte,
	) (int, error)
}
