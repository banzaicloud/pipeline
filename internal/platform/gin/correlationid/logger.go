package correlationid

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const correlationIdField = "correlation-id"

// Logger returns a new logger instance with a correlation ID in it.
func Logger(logger logrus.FieldLogger, ctx *gin.Context) logrus.FieldLogger {
	cid := ctx.GetString(ContextKey)

	if cid == "" {
		return logger
	}

	return logger.WithField(correlationIdField, cid)
}
