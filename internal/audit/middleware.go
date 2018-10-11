// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/spotguide"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

type closeableBuffer struct {
	*bytes.Buffer
}

func (*closeableBuffer) Close() error {
	return nil
}

// LogWriter instance is a Gin Middleware which logs all request data into MySQL audit_events table.
func LogWriter(
	skipPaths []string,
	whitelistedHeaders []string,
	db *gorm.DB,
	logger logrus.FieldLogger,
) gin.HandlerFunc {
	skip := map[string]struct{}{}

	for _, path := range skipPaths {
		skip[path] = struct{}{}
	}

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
				logger.Errorln(err)

				return
			}

			if written != c.Request.ContentLength {
				c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Failed to copy request body correctly"))
				logger.Errorln(err)

				return
			}

			rawBody := bodyBuffer.Bytes()
			c.Request.Body = bodyBuffer

			// Filter out sensitive data from body
			var body *string

			if len(rawBody) > 0 {

				if !json.Valid(rawBody) {
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error ": "invalid JSON in body"})
					return
				}

				if strings.Contains(path, "/secrets") || strings.Contains(path, "/spotguides") {

					var request struct {
						*secret.CreateSecretRequest
						*spotguide.LaunchRequest
					}

					err := json.Unmarshal(rawBody, &request)
					if err != nil {
						c.AbortWithError(http.StatusInternalServerError, err)
						logger.Errorln(err)

						return
					}

					newBody := rawBody

					if request.CreateSecretRequest != nil {
						newBody, err = json.Marshal(&request.CreateSecretRequest)
					} else if request.LaunchRequest != nil {
						newBody, err = json.Marshal(&request.LaunchRequest)
					}

					if err != nil {
						c.AbortWithError(http.StatusInternalServerError, err)
						logger.Errorln(err)

						return
					}

					newBodyString := string(newBody)
					body = &newBodyString

				} else {

					newBodyString := string(rawBody)
					body = &newBodyString
				}
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
				logger.Errorln(err)

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
				logger.Errorln(err)

				return
			}
		}
	}
}
