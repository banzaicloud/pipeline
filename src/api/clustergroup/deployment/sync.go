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

package deployment

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	gutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/src/auth"
)

// @Summary Synchronize Cluster Group Deployment
// @Description install / upgrade deployment to target clusters where deployment is not found or has wrong
// 	version/values (like somebody deleted, updated the deployment on that given cluster using Single Cluster Deployment API),
// 	deletes deployment from target clusters which are not member of a cluster group anymore
// @Tags clustergroup deployments
// @Accept json
// @Produce json
// @Param orgid path uint true "Organization ID"
// @Param clusterGroupId path uint true "Cluster Group ID"
// @Param deploymentName path string true "release name of a cluster group deployment"
// @Success 202 {object} deployment.TargetClusterStatus "Multi-cluster deployment has been synced successfully. All upgrade / install operations on all targeted clusters returned with no errors."
// @Success 207 {object} common.ErrorResponse "Partial failure, meaning that there was as least one failure on one of the target clusters"
// @Failure 400 {object} common.ErrorResponse Deployment Not Found
// @Router /api/v1/orgs/{orgid}/clustergroups/{clusterGroupId}/deployments/{deploymentName}/sync [put]
// @Security bearerAuth
func (n *API) Sync(c *gin.Context) {
	ctx := gutils.Context(context.Background(), c)

	name := c.Param("name")
	n.logger.Infof("sync cluster group deployment: [%s]", name)

	clusterGroupID, ok := gutils.UintParam(c, "id")
	if !ok {
		return
	}

	orgID := auth.GetCurrentOrganization(c.Request).ID
	clusterGroup, err := n.clusterGroupManager.GetClusterGroupByID(ctx, clusterGroupID, orgID)
	if err != nil {
		n.errorHandler.Handle(c, err)
		return
	}

	organization, err := auth.GetOrganizationById(clusterGroup.OrganizationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting organization",
			Error:   err.Error(),
		})
		return
	}

	response, err := n.deploymentManager.SyncDeployment(clusterGroup, organization.Name, name)
	if err != nil {
		n.errorHandler.Handle(c, err)
		return
	}

	if !n.returnOperationErrorsIfAny(c, response, name) {
		response, err := n.deploymentManager.GetDeployment(clusterGroup, name)
		if err != nil {
			n.errorHandler.Handle(c, err)
			return
		}
		c.JSON(http.StatusAccepted, response.TargetClustersStatus)
	}
}
