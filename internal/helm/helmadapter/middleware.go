// Copyright © 2020 Banzai Cloud
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

package helmadapter

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/banzaicloud/pipeline/internal/helm"
)

// envEnsurerMiddleware component providing middleware logic for reconciling persisted customer helm repositories
type envEnsurerMiddleware struct {
	envEnsurer helm.EnvResolver // must be an ensuring envresolver
	logger     Logger
}

// NewHelmEnvEnsurerMiddleware
func NewHelmEnvEnsurerMiddleware(envResolver helm.EnvResolver, logger Logger) envEnsurerMiddleware {
	return envEnsurerMiddleware{
		envEnsurer: envResolver,
		logger:     logger,
	}
}

func (e envEnsurerMiddleware) Middleware(c *gin.Context) {
	orgIDStr := c.Param("orgid")
	orgID, err := strconv.ParseInt(orgIDStr, 10, 32)
	if err != nil {
		return
	}

	if orgID < 1 {
		// there's no org id found in the URL, do nothing
		return
	}

	// make sure the helm env is created and persisted repos are restored
	if _, err := e.envEnsurer.ResolveHelmEnv(c.Request.Context(), uint(orgID)); err != nil {
		e.logger.Warn("failed to ensure helm env", map[string]interface{}{"orgID": orgID})
	}
}
