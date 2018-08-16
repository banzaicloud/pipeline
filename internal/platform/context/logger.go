package context

import (
	"context"

	"github.com/sirupsen/logrus"
)

const correlationIdField = "correlation-id"

// LoggerWithCorrelationID returns a new logger with a correlation ID in it's context.
func LoggerWithCorrelationID(ctx context.Context, logger logrus.FieldLogger) logrus.FieldLogger {
	cid, ok := ctx.Value(contextKeyCorrelationId).(string)
	if !ok || cid == "" {
		return logger
	}

	return logger.WithField(correlationIdField, cid)
}
