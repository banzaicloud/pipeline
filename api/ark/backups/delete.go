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
	"github.com/banzaicloud/pipeline/internal/ark/api"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
	"github.com/banzaicloud/pipeline/internal/platform/gin/utils"
)

// Delete deletes an ARK backup
func Delete(c *gin.Context) {
	logger := correlationid.Logger(common.Log, c)

	backupID, ok := ginutils.UintParam(c, IDParamName)
	if !ok {
		return
	}

	logger = logger.WithField("backup", backupID)
	logger.Info("deleting backup")

	err := common.GetARKService(c.Request).GetClusterBackupsService().DeleteByID(backupID)
	if err != nil {
		err = emperror.Wrap(err, "could not delete backup")
		logger.Error(err.Error())
		common.ErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, &api.DeleteBackupResponse{
		ID:     backupID,
		Status: http.StatusOK,
	})
}
