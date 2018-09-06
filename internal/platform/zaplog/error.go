package zaplog

import (
	"github.com/goph/emperror"
	"go.uber.org/zap"
)

// LogError logs an error.
func LogError(logger *zap.Logger, err error) {
	errCtx := emperror.Context(err)
	if len(errCtx) > 0 {
		for i := 0; i < len(errCtx); i += 2 {
			key := errCtx[i].(string)

			logger = logger.With(zap.Any(key, errCtx[i+1]))
		}
	}

	logger.Error(err.Error())
}
