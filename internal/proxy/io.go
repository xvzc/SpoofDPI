package proxy

import (
	"io"
)

// closeConns safely closes one or more io.Closer (like net.Conn).
// It is nil-safe and intentionally ignores errors from Close(),
// which is a common pattern in defer statements where handling the
// error is not feasible or desired.
func closeConns(closers ...io.Closer) {
	for _, c := range closers {
		if c != nil {
			// Intentionally ignore the error.
			_ = c.Close()
		}
	}
}
