package netutil

import (
	"errors"
	"net"
	"syscall"
)

var ErrBlocked = errors.New("request blocked")

func IsConnectionResetByPeer(err error) bool {
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		var sysErr syscall.Errno
		if errors.As(opErr.Err, &sysErr) {
			return sysErr == syscall.ECONNRESET
		}
	}

	return false
}
