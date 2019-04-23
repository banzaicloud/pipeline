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

package backups

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"

	"github.com/banzaicloud/pipeline/api/ark/common"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/config"
	arkClusterManager "github.com/banzaicloud/pipeline/internal/ark/clustermanager"
	"github.com/banzaicloud/pipeline/internal/ark/sync"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
)

// Sync syncs ARK backups for the organization
func (b *orgBackups) Sync(c *gin.Context) {
	logger := correlationid.Logger(common.Log, c)
	logger.Info("syncing backups")

	org := auth.GetCurrentOrganization(c.Request)
	err := sync.NewBackupsSyncService(org, config.DB(), logger).SyncBackups(arkClusterManager.New(b.clusterManager))
	if err != nil {
		err = emperror.WrapWith(err, "could not sync org backups", "orgName", org.Name)
		common.ErrorHandler.Handle(err)
		common.ErrorResponse(c, err)
		return
	}

	c.Status(http.StatusOK)
}
