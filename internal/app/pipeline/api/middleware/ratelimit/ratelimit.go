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

package ratelimit

import (
	"github.com/didip/tollbooth"
	"github.com/gin-gonic/gin"

	"github.com/banzaicloud/pipeline/auth"
)

// NewRateLimiterByOrgID creates a middleware to rate-limit requests by organization ID
func NewRateLimiterByOrgID(max float64) gin.HandlerFunc {
	limiter := tollbooth.NewLimiter(max, nil)

	return func(c *gin.Context) {
		orgID := auth.GetCurrentOrganization(c.Request).ID

		httpError := tollbooth.LimitByKeys(limiter, []string{string(orgID)})
		if httpError != nil {
			c.Data(httpError.StatusCode, limiter.GetMessageContentType(), []byte(httpError.Message))
			c.Abort()
			return
		}

		c.Next()
	}
}
