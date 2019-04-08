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

package restores

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/api/ark/common"
	"github.com/banzaicloud/pipeline/internal/ark"
	"github.com/banzaicloud/pipeline/internal/ark/sync"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
)

// Sync synchronizes ARK restores
func Sync(c *gin.Context) {
	logger := correlationid.Logger(common.Log, c)
	logger.Info("syncing restores")

	arkSvc := common.GetARKService(c.Request)
	err := syncRestores(arkSvc, logger)
	if err != nil {
		err = emperror.Wrap(err, "could not sync restores")
		common.ErrorHandler.Handle(err)
		common.ErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, nil)
}

func syncRestores(arkSvc *ark.Service, logger logrus.FieldLogger) error {
	restoresSyncSvc := sync.NewRestoresSyncService(arkSvc.GetOrganization(), arkSvc.GetDB(), logger)
	err := restoresSyncSvc.SyncRestoresForCluster(arkSvc.GetCluster())
	if err != nil {
		return emperror.Wrap(err, "could not sync restores")
	}

	return nil
}
