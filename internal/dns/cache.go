package dns

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/datastruct/cache"
	"github.com/xvzc/SpoofDPI/internal/logging"
)

// CacheResolver is a decorator that adds caching functionality to another Resolver.
type CacheResolver struct {
	logger zerolog.Logger

	ttlCache cache.Cache // Owns the cache
	next     Resolver    // The "worker" resolver
}

// NewCacheResolver wraps a "worker" resolver with a cache.
func NewCacheResolver(
	logger zerolog.Logger,
	cache cache.Cache,
	next Resolver, // The resolver to be wrapped (e.g., GeneralResolver)
) *CacheResolver {
	return &CacheResolver{
		logger:   logger,
		ttlCache: cache,
		next:     next,
	}
}

func (cr *CacheResolver) Info() []ResolverInfo {
	info := cr.next.Info()
	info[0].Cached = CachedStatus{true}
	return info
}

// Resolve implements the Resolver interface.
// This is where all the cache checking logic resides.
func (cr *CacheResolver) Resolve(
	ctx context.Context,
	domain string,
	qTypes []uint16,
) (*RecordSet, error) {
	logger := logging.WithLocalScope(cr.logger, ctx, "cache")
	// 1. [Cache Read]
	if item, ok := cr.ttlCache.Get(domain); ok {
		logger.Trace().Msgf("hit")
		return item.(*RecordSet), nil
	}

	// 2. [Cache Miss]
	//    Delegate the actual network request to 'r.next' (the worker).
	logger.Trace().Str("next", cr.next.Info()[0].Name).Msgf("miss")
	rSet, err := cr.next.Resolve(ctx, domain, qTypes)
	if err != nil {
		return nil, err
	}

	// 3. [Cache Write]
	// (Assuming the actual TTL is parsed from the DNS response)
	// realTTL := 5 * time.Second
	logger.Trace().
		Int("len", rSet.Count()).
		Uint32("ttl", rSet.TTL()).
		Msg("set")

	_ = cr.ttlCache.Set(
		domain,
		rSet,
		cache.Options().WithTTL(time.Duration(rSet.TTL())*time.Second),
	)

	return rSet, nil
}
