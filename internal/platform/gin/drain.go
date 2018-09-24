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

package gin

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
)

// DrainModeMiddleware prevents write operations from succeeding.
type DrainModeMiddleware struct {
	enabled bool
	mu      sync.RWMutex

	errorHandler emperror.Handler
}

// NewDrainModeMiddleware returns a new DrainModeMiddleware instance.
func NewDrainModeMiddleware(errorHandler emperror.Handler) *DrainModeMiddleware {
	return &DrainModeMiddleware{
		errorHandler: errorHandler,
	}
}

// Middleware implements the gin handler for this middleware.
func (m *DrainModeMiddleware) Middleware(c *gin.Context) {
	if c.Request.URL.Path == "/-/drain" {
		clientIP := c.ClientIP()

		if clientIP != "127.0.0.1" && clientIP != "::1" {
			m.errorHandler.Handle(emperror.With(
				errors.New("Client cannot set drain mode"),
				"client_ip", clientIP,
			))

			c.Next()

			return
		}

		m.mu.Lock()

		switch c.Request.Method {
		case http.MethodPost:
			m.enabled = true

		case http.MethodDelete:
			m.enabled = false
		}

		m.mu.Unlock()

		c.AbortWithStatus(http.StatusOK)

		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.enabled && isWriteOperation(c) {
		c.AbortWithStatusJSON(
			http.StatusServiceUnavailable,
			map[string]string{
				"code":    "503",
				"message": "service is in maintenance mode",
			},
		)

		return
	}

	c.Next()
}

func isWriteOperation(c *gin.Context) bool {
	return c.Request.Method == http.MethodPost ||
		c.Request.Method == http.MethodPut ||
		c.Request.Method == http.MethodPatch ||
		c.Request.Method == http.MethodDelete
}
