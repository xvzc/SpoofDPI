package cache

import (
	"fmt"
	"testing"
	"time"
)

type dummyIPKey [16]byte

func generateDummyIPKey(i int) dummyIPKey {
	var k dummyIPKey
	k[0] = byte(i)
	k[1] = byte(i >> 8)
	return k
}

func BenchmarkCacheKeys(b *testing.B) {
	strCache := NewTTLCache[string](TTLCacheAttrs{
		NumOfShards:     1,
		CleanupInterval: time.Minute,
	})

	ipCache := NewTTLCache[dummyIPKey](TTLCacheAttrs{
		NumOfShards:     1,
		CleanupInterval: time.Minute,
	})

	b.Run("TTLCache_StringKey", func(b *testing.B) {
		var keys []string
		for i := 0; i < b.N; i++ {
			keys = append(keys, "192.168.0."+fmt.Sprint(i))
		}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			key := keys[i]
			strCache.Store(key, 1, nil)
			strCache.Fetch(key)
		}
	})

	b.Run("TTLCache_GenericStructKey", func(b *testing.B) {
		var keys []dummyIPKey
		for i := 0; i < b.N; i++ {
			keys = append(keys, generateDummyIPKey(i))
		}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			key := keys[i]
			ipCache.Store(key, 1, nil)
			ipCache.Fetch(key)
		}
	})
}
