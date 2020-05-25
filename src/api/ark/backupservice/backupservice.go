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

package backupservice

import (
	"github.com/gin-gonic/gin"

	"github.com/banzaicloud/pipeline/internal/helm"

	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/src/api/ark/common"
)

// AddRoutes adds ARK backups related API routes
func AddRoutes(group *gin.RouterGroup, helmService helm.UnifiedReleaser) {
	group.Use(common.ARKMiddleware(global.DB(), common.Log))
	group.HEAD("/status", StatusDeprecated)
	group.GET("/status", Status)
	group.POST("/enable", Enable(helmService))
	group.POST("/disable", Disable(helmService))
}
