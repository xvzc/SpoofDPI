package dns

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/datastruct"
)

// CacheResolver is a decorator that adds caching functionality to another Resolver.
type CacheResolver struct {
	cache  *datastruct.TTLCache[RecordSet] // Owns the cache
	next   Resolver                        // The "worker" resolver
	logger zerolog.Logger
}

// NewCacheResolver wraps a "worker" resolver with a cache.
func NewCacheResolver(
	cache *datastruct.TTLCache[RecordSet],
	next Resolver, // The resolver to be wrapped (e.g., GeneralResolver)
	logger zerolog.Logger,
) *CacheResolver {
	return &CacheResolver{
		logger: logger,
		cache:  cache,
		next:   next,
	}
}

func (cr *CacheResolver) String() string {
	return fmt.Sprintf("cached %s", cr.next)
}

// Resolve implements the Resolver interface.
// This is where all the cache checking logic resides.
func (cr *CacheResolver) Resolve(
	ctx context.Context,
	domain string,
) (RecordSet, error) {
	logger := cr.logger.With().Ctx(ctx).Logger()
	// 1. [Cache Read]
	if item, ok := cr.cache.Get(domain); ok {
		cr.logger.Debug().Ctx(ctx).Msgf("cache hit for %s", domain)
		return item, nil
	}

	// 2. [Cache Miss]
	//    Delegate the actual network request to 'r.next' (the worker).
	logger.Debug().Msgf("cache miss for %s, resolving via next resolver...", domain)
	rSet, err := cr.next.Resolve(ctx, domain)
	if err != nil {
		return RecordSet{addrs: []net.IPAddr{}, ttl: 0}, err
	}

	// 3. [Cache Write]
	// (Assuming the actual TTL is parsed from the DNS response)
	// realTTL := 5 * time.Second
	logger.Debug().Msgf("caching %d records for %s, ttl: %d",
		rSet.Counts(), domain, rSet.TTL(),
	)

	cr.cache.Set(domain, rSet, time.Duration(rSet.TTL())*time.Second)

	return rSet, nil
}
