package dns

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/datastruct"
)

// CacheResolver는 다른 Resolver를 감싸서 캐시 기능을 추가하는 데코레이터입니다.
type CacheResolver struct {
	cache  *datastruct.TTLCache[RecordSet] // 캐시 소유
	next   Resolver                        // "일꾼" 리졸버
	logger zerolog.Logger
}

// NewCacheResolver는 "일꾼" 리졸버를 인자로 받아 캐시로 감쌉니다.
func NewCacheResolver(
	cache *datastruct.TTLCache[RecordSet],
	next Resolver, // 감쌀 대상 (예: GeneralResolver)
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

// Resolve는 Resolver 인터페이스를 구현합니다.
// ★★★ 캐시 중복 로직이 모두 여기에 있습니다 ★★★
func (cr *CacheResolver) Resolve(
	ctx context.Context,
	domain string,
) (RecordSet, error) {
	logger := cr.logger.With().Ctx(ctx).Logger()
	// 1. [캐시 읽기]
	if item, ok := cr.cache.Get(domain); ok {
		cr.logger.Debug().Ctx(ctx).Msgf("cache hit for %s", domain)
		return item, nil
	}

	// 2. [캐시 미스]
	//    'r.next' (일꾼)에게 실제 네트워크 요청을 위임합니다.
	logger.Debug().Msgf("cache miss for %s, resolving via next resolver...", domain)
	rSet, err := cr.next.Resolve(ctx, domain)
	if err != nil {
		return RecordSet{addrs: []net.IPAddr{}, ttl: 0}, err
	}

	// 3. [캐시 쓰기]
	// (DNS 응답에서 실제 TTL을 파싱했다고 가정)
	// realTTL := 5 * time.Second
	logger.Debug().Msgf("caching %d records for %s, ttl: %d",
		rSet.Counts(), domain, rSet.TTL(),
	)

	cr.cache.Set(domain, rSet, time.Duration(rSet.TTL())*time.Second)

	return rSet, nil
}
