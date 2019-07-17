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

	"emperror.dev/emperror"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/banzaicloud/pipeline/api/ark/common"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
)

// StatusDeprecated gets an ARK backup deployment status by trying to create ARK client
// Deprecated: use GET method instead
func StatusDeprecated(c *gin.Context) {
	logger := correlationid.Logger(common.Log, c)
	logger.Info("checking ARK deployment status")

	schedulesSvc := common.GetARKService(c.Request).GetSchedulesService()
	_, err := schedulesSvc.List()
	if err != nil {
		err = errors.New("backup service not deployed")
		pkgCommon.ErrorResponseWithStatus(c, http.StatusNotFound, err)
		return
	}

	c.Status(http.StatusOK)
}

// Status gets an ARK backup deployment status by trying to create ARK client
func Status(c *gin.Context) {
	logger := correlationid.Logger(common.Log, c)
	logger.Info("checking ARK deployment status")

	schedulesSvc := common.GetARKService(c.Request).GetSchedulesService()
	_, err := schedulesSvc.List()
	if err != nil {
		err = emperror.Wrap(err, "backup service not deployed")
		common.ErrorHandler.Handle(err)
		statusEnabledResponse(c, false)
		return
	}

	statusEnabledResponse(c, true)
}

func statusEnabledResponse(c *gin.Context, enabled bool) {
	c.JSON(http.StatusOK, gin.H{
		"enabled": enabled,
	})
}
