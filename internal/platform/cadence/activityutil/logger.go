package activityutil

import (
	"context"

	pipelineCtx "github.com/banzaicloud/pipeline/internal/platform/context"
	"go.uber.org/cadence/activity"
	"go.uber.org/zap"
)

// GetLogger returns a logger that can be used in an activity.
func GetLogger(ctx context.Context) *zap.Logger {
	logger := activity.GetLogger(ctx)

	cid := pipelineCtx.CorrelationID(ctx)
	if cid != "" {
		logger = logger.With(zap.String("correlation-id", cid))
	}

	return logger
}
