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

	"github.com/banzaicloud/pipeline/auth"
	pkgDep "github.com/banzaicloud/pipeline/internal/clustergroup/deployment"
	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
)

// @Summary Update Cluster Group Deployment
// @Description updates a cluster group deployment, installs or upgrades deployment on each member cluster accordingly
// @Tags clustergroup deployments
// @Accept json
// @Produce json
// @Param orgid path uint true "Organization ID"
// @Param clusterGroupId path uint true "Cluster Group ID"
// @Param deploymentName path string true "release name of a cluster group deployment"
// @Param deployment body deployment.ClusterGroupDeployment true "Deployment Update Request"
// @Success 202 {object} deployment.CreateUpdateDeploymentResponse
// @Failure 400 {object} common.ErrorResponse
// @Failure 404 {object} common.ErrorResponse
// @Router /api/v1/orgs/{orgid}/clustergroups/{clusterGroupId}/deployments/{deploymentName} [put]
// @Security bearerAuth
func (n *API) Upgrade(c *gin.Context) {
	ctx := ginutils.Context(context.Background(), c)

	name := c.Param("name")

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

	organization, err := auth.GetOrganizationById(clusterGroup.OrganizationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error  getting organization",
			Error:   err.Error(),
		})
		return
	}
	var deployment *pkgDep.ClusterGroupDeployment
	if err := c.ShouldBindJSON(&deployment); err != nil {
		n.errorHandler.Handle(c, c.Error(err).SetType(gin.ErrorTypeBind))
		return
	}

	deployment.ReleaseName = name

	targetClusterStatus, err := n.deploymentManager.UpdateDeployment(clusterGroup, organization.Name, deployment)
	if err != nil {
		n.errorHandler.Handle(c, err)
		return
	}

	response := pkgDep.CreateUpdateDeploymentResponse{
		ReleaseName:    deployment.ReleaseName,
		TargetClusters: targetClusterStatus,
	}

	c.JSON(http.StatusAccepted, response)
	return
}
