package appctx

import (
	"context"
	"math/rand"
	"strings"
)

// We define unexported key types to prevent key collisions with other packages.
type (
	scopeCtxKey          struct{}
	traceIDCtxKey        struct{}
	patternMatchedCtxKey struct{}
)

// WithScope returns a new context carrying the given scope string.
func WithScope(ctx context.Context, scope string) context.Context {
	return context.WithValue(ctx, scopeCtxKey{}, scope)
}

// ScopeFrom extracts a scope string from the context, if one exists.
func ScopeFrom(ctx context.Context) (string, bool) {
	scope, ok := ctx.Value(scopeCtxKey{}).(string)
	return scope, ok
}

// WithNewTraceID ensures a trace ID is present in the context.
// If one does not exist, it generates a new random trace ID and returns
// a new context carrying it.
// If one already exists, it returns the original context unmodified.
func WithNewTraceID(ctx context.Context) context.Context {
	// Check if a traceId already exists.
	if _, ok := TraceIDFrom(ctx); ok {
		// If it already exists, do nothing.
		return ctx
	}
	// If it does not exist, create a new one.
	return context.WithValue(ctx, traceIDCtxKey{}, generateTraceId())
}

// TraceIDFrom extracts a trace ID string from the context, if one exists.
func TraceIDFrom(ctx context.Context) (string, bool) {
	traceId, ok := ctx.Value(traceIDCtxKey{}).(string)
	if ok {
		return traceId, true
	}
	return "", false
}

func WithPatternMatched(ctx context.Context, patternMatched bool) context.Context {
	return context.WithValue(ctx, patternMatchedCtxKey{}, patternMatched)
}

func PatternMatchedFrom(ctx context.Context) (bool, bool) {
	patternMatched, ok := ctx.Value(patternMatchedCtxKey{}).(bool)
	if ok {
		return patternMatched, true
	}

	return false, false
}

// generateTraceId creates a new random trace ID.
// This logic remains unexported as it's an implementation detail.
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
			r += 0x27 // 'a' - 10
		}
		sb.WriteByte(r + 0x30) // '0'
		if i&7 == 7 && i != 31 {
			sb.WriteByte(0x2D) // '-'
		}
	}
	return sb.String()
}
