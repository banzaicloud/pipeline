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
	"io"
	"io/ioutil"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
	"github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/secret"
	"github.com/banzaicloud/pipeline/src/spotguide"
)

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
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Log only when path is not being skipped
		if _, ok := skip[path]; ok {
			return
		}

		// Copy request body into a new buffer, so other handlers can use it safely
		bodyBuffer := bytes.NewBuffer(nil)

		if _, err := io.Copy(bodyBuffer, c.Request.Body); err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
			logger.Errorf("audit: failed to copy body: %v", err)

			return
		}

		// We can close the old Body right now, it is fully read
		_ = c.Request.Body.Close()

		rawBody := bodyBuffer.Bytes()
		c.Request.Body = ioutil.NopCloser(bodyBuffer)

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
					c.AbortWithStatusJSON(http.StatusBadRequest, common.ErrorResponse{
						Code:    http.StatusBadRequest,
						Message: "Error during binding",
						Error:   err.Error(),
					})
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
					c.AbortWithStatusJSON(http.StatusBadRequest, common.ErrorResponse{
						Code:    http.StatusBadRequest,
						Message: "Error during binding",
						Error:   err.Error(),
					})
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

		correlationID := c.GetString(correlationid.ContextKey)
		clientIP := c.ClientIP()
		method := c.Request.Method
		userAgent := c.Request.UserAgent()

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
			_ = c.AbortWithError(http.StatusInternalServerError, err)
			logger.Errorf("audit: failed to marshal headers: %v", err)

			return
		}

		event := AuditEvent{
			Time:          start,
			CorrelationID: correlationID,
			ClientIP:      clientIP,
			UserAgent:     userAgent,
			UserID:        userID,
			Method:        method,
			Path:          path,
			Body:          body,
			Headers:       string(headers),
		}

		if err := db.Save(&event).Error; err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
			logger.Errorf("audit: failed to write request to db: %v", err)

			return
		}

		c.Next() // process request

		user = auth.GetCurrentUser(c.Request)
		if user != nil {
			userID = user.ID
		}

		responseEvent := AuditEvent{
			UserID:       userID,
			StatusCode:   c.Writer.Status(),
			ResponseSize: c.Writer.Size(),
			ResponseTime: int(time.Since(start).Nanoseconds() / 1000 / 1000), // ms
		}

		if c.IsAborted() {
			if marshalled, err := json.Marshal(c.Errors); err != nil {
				logger.Errorf("audit: failed to marshal c.Errors: %v", err)
			} else {
				errors := string(marshalled)
				responseEvent.Errors = &errors
			}
		}

		if err := db.Model(&event).Updates(responseEvent).Error; err != nil {
			logger.Errorf("audit: failed to write response details: %v", err)
		}
	}
}
