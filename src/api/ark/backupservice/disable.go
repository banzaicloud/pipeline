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
	"net/http"

	"emperror.dev/errors"
	"github.com/gin-gonic/gin"

	"github.com/banzaicloud/pipeline/internal/ark/api"
	"github.com/banzaicloud/pipeline/internal/helm"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
	"github.com/banzaicloud/pipeline/src/api/ark/common"
)

// Disable removes ARK deployment from the cluster
func Disable(service interface{}) func(c *gin.Context) {
	return func(c *gin.Context) {
		logger := correlationid.LogrusLogger(common.Log, c)
		logger.Info("removing backup service from cluster")

		svc := common.GetARKService(c.Request).GetDeploymentsService()

		var err error
		if is2Service, ok := service.(api.Service); ok {
			err = svc.Deactivate(is2Service)
		} else if helmReleaser, ok := service.(helm.UnifiedReleaser); ok {
			err = svc.Remove(helmReleaser)
		}

		if err != nil {
			err = errors.WrapIf(err, "could not remove backup service")
			common.ErrorHandler.Handle(err)
			common.ErrorResponse(c, err)
			return
		}

		c.JSON(http.StatusOK, api.DisableBackupServiceResponse{
			Status: http.StatusOK,
		})
	}
}
