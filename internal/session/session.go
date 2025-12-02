package session

import (
	"context"
	"math/rand/v2"
	"unsafe"
)

// We define unexported key types to prevent key collisions with other packages.
type (
	traceIDCtxKey        struct{}
	policyIncludedCtxKey struct{}
	shouldExploitCtxKey  struct{}
	remoteInfoCtxKey     struct{}
)

// WithNewTraceID ensures a trace ID is present in the context.
// If one does not exist, it generates a new random trace ID and returns
// a new context carrying it.
// If one already exists, it returns the original context unmodified.
func WithNewTraceID(ctx context.Context) context.Context {
	// Check if a traceID already exists.
	if _, ok := TraceIDFrom(ctx); ok {
		// If it already exists, do nothing.
		return ctx
	}
	// If it does not exist, create a new one.
	return context.WithValue(ctx, traceIDCtxKey{}, generateTraceID())
}

// TraceIDFrom extracts a trace ID string from the context, if one exists.
func TraceIDFrom(ctx context.Context) (string, bool) {
	traceID, ok := ctx.Value(traceIDCtxKey{}).(string)
	if ok {
		return traceID, true
	}
	return "", false
}

// WithRemoteInfo returns a new context carrying the given domain name string.
func WithRemoteInfo(ctx context.Context, domain string) context.Context {
	return context.WithValue(ctx, remoteInfoCtxKey{}, domain)
}

// RemoteInfoFrom extracts a domain name string from the context, if one exists.
func RemoteInfoFrom(ctx context.Context) (string, bool) {
	domain, ok := ctx.Value(remoteInfoCtxKey{}).(string)
	return domain, ok
}

func WithPolicyIncluded(ctx context.Context, patternMatched bool) context.Context {
	return context.WithValue(ctx, policyIncludedCtxKey{}, patternMatched)
}

func PolicyIncludedFrom(ctx context.Context) (bool, bool) {
	patternMatched, ok := ctx.Value(policyIncludedCtxKey{}).(bool)

	return patternMatched, ok
}

func WithShouldExploit(ctx context.Context, shouldExploit bool) context.Context {
	return context.WithValue(ctx, shouldExploitCtxKey{}, shouldExploit)
}

func ShouldExploitFrom(ctx context.Context) (bool, bool) {
	shouldExploit, ok := ctx.Value(shouldExploitCtxKey{}).(bool)

	return shouldExploit, ok
}

// generateTraceID creates a new random trace ID.
// This logic remains unexported as it's an implementation detail.
func generateTraceID() string {
	// 16 hex chars require 16 bytes (each hex char is 1 byte).
	b := make([]byte, 16)

	// We use a 64-bit (8 byte) random value, which is encoded as 16 hex characters.
	q := rand.Uint64()

	// iterate from last index (15) down to 0
	for i := 15; i >= 0; i-- {
		r := uint8(q & 0xF)
		q >>= 4
		if r > 9 {
			r += 0x27
		}
		b[i] = r + 0x30
	}

	return unsafe.String(unsafe.SliceData(b), 16)
}
