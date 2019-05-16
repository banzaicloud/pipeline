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
	"fmt"
	"net/http"

	pkgDep "github.com/banzaicloud/pipeline/internal/clustergroup/deployment"
	"github.com/gin-gonic/gin"

	"github.com/banzaicloud/pipeline/auth"
	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
)

// @Summary Create Cluster Group Deployment
// @Description creates a new cluster group deployment, installs or upgrades deployment on each member cluster accordingly
// @Tags clustergroup deployments
// @Accept json
// @Produce json
// @Param orgid path uint true "Organization ID"
// @Param clusterGroupId path uint true "Cluster Group ID"
// @Param deployment body deployment.ClusterGroupDeployment true "Deployment Create Request"
// @Success 201 {object} deployment.CreateUpdateDeploymentResponse
// @Failure 400 {object} common.ErrorResponse
// @Failure 404 {object} common.ErrorResponse
// @Router /api/v1/orgs/{orgid}/clustergroups/{clusterGroupId}/deployments [post]
// @Security bearerAuth
func (n *API) Create(c *gin.Context) {
	ctx := ginutils.Context(context.Background(), c)

	clusterGroupID, ok := ginutils.UintParam(c, "id")
	if !ok {
		return
	}

	organization := auth.GetCurrentOrganization(c.Request)
	if organization == nil {
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Organization not found",
			Error:   "Organization not found",
		})
		return
	}

	clusterGroup, err := n.clusterGroupManager.GetClusterGroupByID(ctx, clusterGroupID, organization.ID)
	if err != nil {
		n.errorHandler.Handle(c, err)
		return
	}

	var deployment *pkgDep.ClusterGroupDeployment
	if err := c.ShouldBindJSON(&deployment); err != nil {
		n.errorHandler.Handle(c, c.Error(err).SetType(gin.ErrorTypeBind))
		return
	}

	if len(deployment.ReleaseName) == 0 {
		deployment.ReleaseName = n.deploymentManager.GenerateReleaseName(clusterGroup)
	}

	if !n.deploymentManager.IsReleaseNameAvailable(clusterGroup, deployment.ReleaseName) {
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("release name %s not available on all target clusters", deployment.ReleaseName),
			Error:   "release name is not unique across target clusters",
		})
		return
	}

	targetClusterStatus, err := n.deploymentManager.CreateDeployment(clusterGroup, organization.Name, deployment)
	if err != nil {
		n.errorHandler.Handle(c, err)
		return
	}

	errMsg := ""
	for _, status := range targetClusterStatus {
		if len(status.Error) > 0 {
			errMsg += fmt.Sprintln("operation failed on cluster " + status.ClusterName + " - " + status.Error)
		}
	}

	if len(errMsg) > 0 {
		c.JSON(http.StatusMultiStatus, pkgCommon.ErrorResponse{
			Code:    http.StatusMultiStatus,
			Message: errMsg,
			Error:   errMsg,
		})
		return
	}

	n.logger.Debug("Release name: ", deployment.ReleaseName)
	response := pkgDep.CreateUpdateDeploymentResponse{
		ReleaseName:    deployment.ReleaseName,
		TargetClusters: targetClusterStatus,
	}
	c.JSON(http.StatusCreated, response)
	return
}
