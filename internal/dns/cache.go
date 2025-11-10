package dns

import (
	"context"
	"net"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/datastruct"
)

// CacheResolver is a decorator that adds caching functionality to another Resolver.
type CacheResolver struct {
	logger zerolog.Logger

	cache *datastruct.TTLCache[RecordSet] // Owns the cache
	next  Resolver                        // The "worker" resolver
}

// NewCacheResolver wraps a "worker" resolver with a cache.
func NewCacheResolver(
	logger zerolog.Logger,
	cache *datastruct.TTLCache[RecordSet],
	next Resolver, // The resolver to be wrapped (e.g., GeneralResolver)
) *CacheResolver {
	return &CacheResolver{
		logger: logger,
		cache:  cache,
		next:   next,
	}
}

func (cr *CacheResolver) Info() []ResolverInfo {
	info := cr.next.Info()
	info[0].Cached = CachedStatus{true}
	return info
}

func (cr *CacheResolver) Route(ctx context.Context) (Resolver, bool) {
	r, ok := cr.next.Route(ctx)
	if !ok {
		return nil, false
	}

	if r != nil {
		return cr, true
	}

	return nil, true
}

// Resolve implements the Resolver interface.
// This is where all the cache checking logic resides.
func (cr *CacheResolver) Resolve(
	ctx context.Context,
	domain string,
	qTypes []uint16,
) (RecordSet, error) {
	logger := cr.logger.With().Ctx(ctx).Logger()
	// 1. [Cache Read]
	if item, ok := cr.cache.Get(domain); ok {
		cr.logger.Debug().Ctx(ctx).Msgf("cache hit; key=%s;", domain)
		return item, nil
	}

	// 2. [Cache Miss]
	//    Delegate the actual network request to 'r.next' (the worker).
	logger.Debug().Msgf("cache miss; key=%s; next=%s;", domain, cr.next.Info()[0].Name)
	rSet, err := cr.next.Resolve(ctx, domain, qTypes)
	if err != nil {
		return RecordSet{addrs: []net.IPAddr{}, ttl: 0}, err
	}

	// 3. [Cache Write]
	// (Assuming the actual TTL is parsed from the DNS response)
	// realTTL := 5 * time.Second
	logger.Debug().Msgf("cache set; key=%s; len=%d;, ttl: %d",
		domain, rSet.Counts(), rSet.TTL(),
	)

	cr.cache.Set(domain, rSet, time.Duration(rSet.TTL())*time.Second)

	return rSet, nil
}
