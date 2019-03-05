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

package backups

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/api/ark/common"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/internal/ark"
	arkClusterManager "github.com/banzaicloud/pipeline/internal/ark/clustermanager"
	"github.com/banzaicloud/pipeline/internal/ark/sync"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
)

// Sync synchronizes ARK backups
func Sync(c *gin.Context) {
	logger := correlationid.Logger(common.Log, c)
	logger.Info("syncing backups")

	err := syncBackups(common.GetARKService(c.Request), logger)
	if err != nil {
		common.ErrorHandler.Handle(err)
		common.ErrorResponse(c, err)
		return
	}

	c.Status(http.StatusOK)
}

func syncOrgBackups(clusterManager *cluster.Manager, org *auth.Organization, db *gorm.DB, logger logrus.FieldLogger) error {
	backupSyncSvc := sync.NewBackupsSyncService(org, db, logger)
	err := backupSyncSvc.SyncBackups(arkClusterManager.New(clusterManager))
	if err != nil {
		return emperror.Wrap(err, "could not sync org backups")
	}

	return nil
}

func syncBackups(arkSvc *ark.Service, logger logrus.FieldLogger) error {
	backupSyncSvc := sync.NewBackupsSyncService(arkSvc.GetOrganization(), arkSvc.GetDB(), logger)
	err := backupSyncSvc.SyncBackupsForCluster(arkSvc.GetCluster())
	if err != nil {
		return emperror.Wrap(err, "could not sync backups")
	}

	return nil
}
