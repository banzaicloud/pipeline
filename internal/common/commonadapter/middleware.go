// Copyright © 2019 Banzai Cloud
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

package commonadapter

import (
	"github.com/gin-gonic/gin"

	"github.com/banzaicloud/pipeline/internal/common"
)

// LoggerInContext returns a gin handler function that associates the specified logger with the request's context
func LoggerInContext(logger common.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		contextWithLogger := ContextWithLogger(c.Request.Context(), logger)
		c.Request = c.Request.WithContext(contextWithLogger)
		c.Next()
	}
}
