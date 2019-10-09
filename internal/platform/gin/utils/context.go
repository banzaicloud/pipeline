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

package ginutils

import (
	"context"

	"github.com/gin-gonic/gin"

	pipelineContext "github.com/banzaicloud/pipeline/internal/platform/context"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
)

// Context returns a new Go context from a Gin context.
func Context(c *gin.Context) context.Context {
	ctx := c.Request.Context()

	if cid := c.GetString(correlationid.ContextKey); cid != "" {
		ctx = pipelineContext.WithCorrelationID(ctx, cid)
	}

	return ctx
}
