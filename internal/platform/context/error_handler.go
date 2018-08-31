package context

import (
	"context"

	"github.com/goph/emperror"
)

// ErrorHandlerWithCorrelationID returns a new error handler with a correlation ID in it's context.
func ErrorHandlerWithCorrelationID(ctx context.Context, errorHandler emperror.Handler) emperror.Handler {
	cid, ok := ctx.Value(contextKeyCorrelationId).(string)
	if !ok || cid == "" {
		return errorHandler
	}

	return emperror.HandlerWith(errorHandler, correlationIdField, cid)
}
