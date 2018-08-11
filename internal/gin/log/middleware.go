package log

import (
	"time"

	"github.com/banzaicloud/pipeline/internal/gin/correlationid"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const correlationIdField = "correlation-id"

// Middleware returns a gin compatible handler.
func Middleware(logger logrus.FieldLogger, notlogged ...string) gin.HandlerFunc {
	var skip map[string]struct{}

	if length := len(notlogged); length > 0 {
		skip = make(map[string]struct{}, length)

		for _, path := range notlogged {
			skip[path] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		// start timer
		start := time.Now()

		// prevent middlewares from faking the request path
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		// Log only when path is not being skipped
		if _, ok := skip[path]; !ok {
			end := time.Now()
			latency := end.Sub(start)

			if raw != "" {
				path = path + "?" + raw
			}

			fields := logrus.Fields{
				"status":     c.Writer.Status(),
				"method":     c.Request.Method,
				"path":       path,
				"ip":         c.ClientIP(),
				"latency":    latency,
				"user-agent": c.Request.UserAgent(),
			}

			if cid := c.GetString(correlationid.ContextKey); cid != "" {
				fields[correlationIdField] = cid
			}

			entry := logger.WithFields(fields)

			if len(c.Errors) > 0 {
				// Append error field if this is an erroneous request.
				entry.Error(c.Errors.String())
			} else {
				entry.Info()
			}
		}
	}
}
