package src

import "context"

type correlationIdKey struct{}

func AddCorrelationID(ctx context.Context) context.Context {
	id := ctx.Value(correlationIdKey{})
	if id != nil {
		return ctx
	}

	return context.WithValue(ctx, correlationIdKey{}, id)
}

func GetCorrelationID(ctx context.Context) any {
	return ctx.Value(correlationIdKey{})
}
