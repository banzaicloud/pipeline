package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/spf13/cast"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/model"
	"github.com/gin-gonic/gin"
)

var log *logrus.Entry = config.Logger().WithField("tag", "Audit")

type closeableBuffer struct {
	*bytes.Buffer
}

func (*closeableBuffer) Close() error {
	return nil
}

// AuditEvent holds all information related to a user interaction
type AuditEvent struct {
	Time       time.Time
	Latency    time.Duration
	ClientIP   string
	UserAgent  string
	Path       string
	Method     string
	UserID     uint
	StatusCode int
	Body       string `sql:"TYPE:json"`
	Comment    string
}

// LogWriter instance is a Gin Middleware which logs all request data into MySQL audit_events table.
func LogWriter(notloggedPaths ...string) gin.HandlerFunc {
	var skip map[string]struct{}

	if length := len(notloggedPaths); length > 0 {
		skip = make(map[string]struct{}, length)

		for _, path := range notloggedPaths {
			skip[path] = struct{}{}
		}
	}

	db := model.GetDB()

	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Copy request body into a new buffer
		bodyBuffer := &closeableBuffer{bytes.NewBuffer(nil)}

		written, err := io.Copy(bodyBuffer, c.Request.Body)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			log.Errorln(err)
			return
		}

		if written != c.Request.ContentLength {
			c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Failed to copy request body correctly"))
			log.Errorln(err)
			return
		}

		rawBody := bodyBuffer.Bytes()
		c.Request.Body = bodyBuffer

		// Process request
		c.Next()

		// Log only when path is not being skipped
		if _, ok := skip[path]; !ok {
			// Stop timer
			end := time.Now()
			latency := end.Sub(start)

			// Filter out sensitive data from body
			var body string
			if strings.Contains(path, "/secrets") && len(rawBody) > 0 {
				data := map[string]interface{}{}
				err := json.Unmarshal(rawBody, &data)
				if err != nil {
					c.AbortWithError(http.StatusInternalServerError, err)
					log.Errorln(err)
					return
				}
				values := cast.ToStringMapString(data["values"])
				for k := range values {
					values[k] = ""
				}
				newBody, err := json.Marshal(data)
				if err != nil {
					c.AbortWithError(http.StatusInternalServerError, err)
					log.Errorln(err)
					return
				}
				body = string(newBody)
			} else {
				body = string(rawBody)
			}

			clientIP := c.ClientIP()
			method := c.Request.Method
			userAgent := c.Request.UserAgent()
			statusCode := c.Writer.Status()
			comment := c.Errors.ByType(gin.ErrorTypePrivate).String()

			if raw != "" {
				path = path + "?" + raw
			}

			user := auth.GetCurrentUser(c.Request)
			var userID uint
			if user != nil {
				userID = user.ID
			}

			event := AuditEvent{
				Time:       start,
				Latency:    latency,
				ClientIP:   clientIP,
				UserAgent:  userAgent,
				UserID:     userID,
				StatusCode: statusCode,
				Method:     method,
				Body:       body,
				Path:       path,
				Comment:    comment,
			}

			err = db.Save(&event).Error
			if err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
				log.Errorln(err)
				return
			}
		}
	}
}
