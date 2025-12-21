package proxy

import (
	"context"
)

type ProxyServer interface {
	ListenAndServe(ctx context.Context, wait chan struct{})
}
