package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/spf13/cast"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/config"
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
	ID         uint      `gorm:"primary_key"`
	Time       time.Time `gorm:"index"`
	ClientIP   string    `gorm:"size:45"`
	UserAgent  string
	Path       string `gorm:"size:8000"`
	Method     string `gorm:"size:7"`
	UserID     uint
	StatusCode int
	Body       *string `gorm:"type:json"`
	Headers    string  `gorm:"type:json"`
}

// LogWriter instance is a Gin Middleware which logs all request data into MySQL audit_events table.
func LogWriter(notloggedPaths []string, whitelistedHeaders []string) gin.HandlerFunc {
	skip := map[string]struct{}{}

	for _, path := range notloggedPaths {
		skip[path] = struct{}{}
	}

	db := config.DB()

	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Log only when path is not being skipped
		if _, ok := skip[path]; !ok {

			// Copy request body into a new buffer, so other handlers can use it safely
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

			// Filter out sensitive data from body
			var body *string
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
				newBodyString := string(newBody)
				body = &newBodyString
			} else if len(rawBody) > 0 {
				newBodyString := string(rawBody)
				body = &newBodyString
			}

			clientIP := c.ClientIP()
			method := c.Request.Method
			userAgent := c.Request.UserAgent()
			statusCode := c.Writer.Status()

			if raw != "" {
				path = path + "?" + raw
			}

			user := auth.GetCurrentUser(c.Request)
			var userID uint
			if user != nil {
				userID = user.ID
			}

			filteredHeaders := http.Header{}
			for _, header := range whitelistedHeaders {
				if values := c.Request.Header[textproto.CanonicalMIMEHeaderKey(header)]; len(values) != 0 {
					filteredHeaders[header] = values
				}
			}
			headers, err := json.Marshal(filteredHeaders)
			if err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
				log.Errorln(err)
				return
			}

			event := AuditEvent{
				Time:       start,
				ClientIP:   clientIP,
				UserAgent:  userAgent,
				UserID:     userID,
				StatusCode: statusCode,
				Method:     method,
				Path:       path,
				Body:       body,
				Headers:    string(headers),
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
