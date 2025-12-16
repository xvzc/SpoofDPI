package proxy

import (
	"context"
)

type Proxy interface {
	ListenAndServe(ctx context.Context, wait chan struct{})
}
