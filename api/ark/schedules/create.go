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
	"k8s.io/apimachinery/pkg/labels"

	"github.com/banzaicloud/pipeline/api/ark/common"
	"github.com/banzaicloud/pipeline/internal/ark/api"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
)

// Create creates an ARK schedule
func Create(c *gin.Context) {
	logger := correlationid.Logger(common.Log, c)
	logger.Info("creating schedule")

	var request api.CreateScheduleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		err = emperror.Wrap(err, "could not parse request")
		common.ErrorHandler.Handle(err)
		common.ErrorResponse(c, err)
		return
	}

	spec := &api.CreateBackupRequest{
		Name:    request.Name,
		Labels:  request.Labels,
		TTL:     request.TTL,
		Options: request.Options,
	}

	svc := common.GetARKService(c.Request)

	if spec.Labels == nil {
		spec.Labels = make(labels.Set, 0)
	}
	spec.Labels[api.LabelKeyDistribution] = string(svc.GetDeploymentsService().GetCluster().GetDistribution())
	spec.Labels[api.LabelKeyCloud] = svc.GetDeploymentsService().GetCluster().GetCloud()

	err := svc.GetSchedulesService().Create(spec, request.Schedule)
	if err != nil {
		err = emperror.Wrap(err, "could not create schedule")
		common.ErrorHandler.Handle(err)
		common.ErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, &api.CreateScheduleResponse{
		Name:   spec.Name,
		Status: http.StatusOK,
	})
}
