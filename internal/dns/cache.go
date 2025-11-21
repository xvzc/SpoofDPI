package dns

import (
	"context"
	"net"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/datastruct/cache"
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

func (cr *CacheResolver) Route(ctx context.Context) Resolver {
	if next := cr.next.Route(ctx); next != nil {
		return cr
	}

	return nil
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
	if item, ok := cr.ttlCache.Get(domain); ok {
		cr.logger.Debug().Ctx(ctx).Msgf("dns cache hit; name=%s;", domain)
		return item.(RecordSet), nil
	}

	// 2. [Cache Miss]
	//    Delegate the actual network request to 'r.next' (the worker).
	logger.Debug().
		Msgf("dns cache miss; name=%s; next=%s;", domain, cr.next.Info()[0].Name)
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

	cr.ttlCache.Set(
		domain,
		rSet,
		cache.Options().
			WithOverride(true).
			WithTTL(time.Duration(rSet.TTL())*time.Second),
	)

	return rSet, nil
}
