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

package schedules

import (
	"net/http"

	"emperror.dev/emperror"
	"github.com/gin-gonic/gin"

	"github.com/banzaicloud/pipeline/api/ark/common"
	arkAPI "github.com/banzaicloud/pipeline/internal/ark/api"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
)

// Delete deletes an ARK schedule
func Delete(c *gin.Context) {
	scheduleName := c.Param("name")

	logger := correlationid.Logger(common.Log, c).WithField("schedule", scheduleName)
	logger.Info("deleting schedule")

	err := common.GetARKService(c.Request).GetSchedulesService().DeleteByName(scheduleName)
	if err != nil {
		err = emperror.Wrap(err, "could not delete schedule")
		common.ErrorHandler.Handle(err)
		common.ErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, &arkAPI.DeleteScheduleResponse{
		Name:   scheduleName,
		Status: http.StatusOK,
	})
}
