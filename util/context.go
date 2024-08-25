package util

import (
	"context"
	"math/rand"
	"strings"
)

type scopeCtxKey struct{}

func GetCtxWithScope(ctx context.Context, scope string) context.Context {
	return context.WithValue(ctx, scopeCtxKey{}, scope)
}

func GetScopeFromCtx(ctx context.Context) (string, bool) {
	if scope, ok := ctx.Value(scopeCtxKey{}).(string); ok {
		return scope, true
	}
	return "", false
}

type traceIdCtxKey struct{}

func GetCtxWithTraceId(ctx context.Context) context.Context {
	return context.WithValue(ctx, traceIdCtxKey{}, generateTraceId())
}

func GetTraceIdFromCtx(ctx context.Context) (string, bool) {
	if traceId, ok := ctx.Value(traceIdCtxKey{}).(string); ok {
		return traceId, true
	}
	return "", false
}

func generateTraceId() string {
	sb := strings.Builder{}
	sb.Grow(35)

	var q uint64
	var r uint8
	for i := 0; i < 32; i++ {
		if i%15 == 0 {
			q = rand.Uint64()
		}
		q, r = q>>4, uint8(q&0xF)
		if r > 9 {
			r += 0x27
		}
		sb.WriteByte(r + 0x30)
		if i&7 == 7 && i != 31 {
			sb.WriteByte(0x2D)
		}
	}
	return sb.String()
}
