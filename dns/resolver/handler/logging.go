package handler

import (
	"context"
	"net"
	"time"

	"github.com/xvzc/SpoofDPI/dns/resolver"
	"github.com/xvzc/SpoofDPI/util"
	"github.com/xvzc/SpoofDPI/util/log"
)

type LoggingHandler struct {
}

const scopeDNS = "DNS"

func NewLoggingHandler() *LoggingHandler {
	return &LoggingHandler{}
}

func (l *LoggingHandler) DoHandle(ctx context.Context, host string, qTypes []uint16, resolver resolver.Resolver) ([]net.IPAddr, error) {
	ctx = util.GetCtxWithScope(ctx, scopeDNS)
	logger := log.GetCtxLogger(ctx)
	logger.Debug().Msgf("resolving %s using %s", host, resolver)
	t := time.Now()
	addrs, err := resolver.Resolve(ctx, host, qTypes)
	if err != nil {
		logger.Debug().Msgf("failed to resolve %s using %s", host, resolver)
		return nil, err
	}
	if len(addrs) > 0 {
		d := time.Since(t).Milliseconds()
		logger.Debug().Msgf("resolved %s from %s in %d ms", addrs[0].String(), host, d)
	}
	return addrs, nil
}
