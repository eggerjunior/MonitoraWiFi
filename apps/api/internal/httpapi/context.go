package httpapi

import (
	"context"

	"egger/api/internal/store"
)

type contextKey int

const (
	correlationIDKey contextKey = iota
	authenticatedUserKey
	authenticatedAgentKey
)

func withCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey, id)
}

func correlationIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(correlationIDKey).(string); ok {
		return v
	}
	return ""
}

func withUser(ctx context.Context, u store.User) context.Context {
	return context.WithValue(ctx, authenticatedUserKey, u)
}

func userFromContext(ctx context.Context) (store.User, bool) {
	u, ok := ctx.Value(authenticatedUserKey).(store.User)
	return u, ok
}

func withAgent(ctx context.Context, a store.Agent) context.Context {
	return context.WithValue(ctx, authenticatedAgentKey, a)
}

func agentFromContext(ctx context.Context) (store.Agent, bool) {
	a, ok := ctx.Value(authenticatedAgentKey).(store.Agent)
	return a, ok
}
