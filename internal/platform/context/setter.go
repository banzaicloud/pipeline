package context

import "context"

// WithCorrelationID returns a new context with the current correlation ID.
func WithCorrelationID(ctx context.Context, cid string) context.Context {
	return context.WithValue(ctx, contextKeyCorrelationId, cid)
}
