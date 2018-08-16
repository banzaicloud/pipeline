package ginutils

import (
	"context"

	pipelineContext "github.com/banzaicloud/pipeline/internal/platform/context"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
	"github.com/gin-gonic/gin"
)

// Context returns a new Go context from a Gin context.
func Context(ctx context.Context, c *gin.Context) context.Context {
	cid := c.GetString(correlationid.ContextKey)

	if cid == "" {
		return ctx
	}

	return pipelineContext.WithCorrelationID(ctx, cid)
}
