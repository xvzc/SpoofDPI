package handler

import (
	"context"
	"fmt"
	"net"

	"github.com/xvzc/SpoofDPI/dns/resolver"
)

type ErrorHandler struct {
}

func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{}
}

func (e *ErrorHandler) DoHandle(ctx context.Context, host string, qTypes []uint16, next resolver.Resolver) ([]net.IPAddr, error) {
	addrs, err := next.Resolve(ctx, host, qTypes)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", next, err)
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("could not resolve %s using %s", host, next)
	}
	return addrs, nil
}
