package util

import "context"

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
