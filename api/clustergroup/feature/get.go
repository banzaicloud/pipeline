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

package feature

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/banzaicloud/pipeline/auth"
	cgroupIAPI "github.com/banzaicloud/pipeline/internal/clustergroup/api"
	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
)

// @Summary Get Feature of Cluster Group
// @Description retrieve info about a cluster group feature and it's status on each member cluster
// @Tags clustergroup features
// @Accept json
// @Produce json
// @Param orgid path uint true "Organization ID"
// @Param clusterGroupId path uint true "Cluster Group ID"
// @Param featureName path string true "Name of the feature"
// @Success 200 {object} api.FeatureResponse
// @Failure 400 {object} common.ErrorResponse Feature Not Found
// @Router /api/v1/orgs/{orgid}/clustergroups/{clusterGroupId}/features/{featureName} [get]
// @Security bearerAuth
func (n *API) Get(c *gin.Context) {
	ctx := ginutils.Context(context.Background(), c)

	clusterGroupID, ok := ginutils.UintParam(c, "id")
	if !ok {
		return
	}

	orgID := auth.GetCurrentOrganization(c.Request).ID
	clusterGroup, err := n.clusterGroupManager.GetClusterGroupByID(ctx, clusterGroupID, orgID)
	if err != nil {
		n.errorHandler.Handle(c, err)
		return
	}

	featureName := c.Param("featureName")
	feature, err := n.clusterGroupManager.GetFeature(*clusterGroup, featureName)
	if err != nil {
		n.errorHandler.Handle(c, err)
		return
	}

	var response cgroupIAPI.FeatureResponse
	response.Name = feature.Name
	response.Enabled = feature.Enabled
	response.ClusterGroup = feature.ClusterGroup
	response.Properties = feature.Properties
	response.ReconcileState = feature.ReconcileState
	response.LastReconcileError = feature.LastReconcileError

	//call feature handler to get statuses
	if feature.Enabled {
		status, err := n.clusterGroupManager.GetFeatureStatus(*feature)
		if err != nil {
			n.errorHandler.Handle(c, err)
			return
		}
		response.Status = status
	}

	c.JSON(http.StatusOK, response)
}
