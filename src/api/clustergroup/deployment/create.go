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

	"emperror.dev/errors"
	"github.com/gin-gonic/gin"

	pkgDep "github.com/banzaicloud/pipeline/internal/clustergroup/deployment"
	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/src/auth"
)

// @Summary Create Cluster Group Deployment
// @Description creates a new cluster group deployment, installs or upgrades deployment on each member cluster accordingly
// @Tags clustergroup deployments
// @Accept json
// @Produce json
// @Param orgid path uint true "Organization ID"
// @Param clusterGroupId path uint true "Cluster Group ID"
// @Param deployment body deployment.ClusterGroupDeployment true "Deployment Create Request"
// @Success 201 {object} deployment.CreateUpdateDeploymentResponse "Multi-cluster deployment has been created successfully. All install / upgrade operations on all targeted clusters returned with no errors."
// @Success 207 {object} common.ErrorResponse "Partial failure, meaning that Multi-cluster deployment has been created successfully, however there was as least one failure on one of the target clusters"
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

	if deployment.Package != nil {
		n.errorHandler.Handle(c, errors.New("deployment using custom chart content is unsupported"))
		return
	}

	if len(deployment.ReleaseName) == 0 {
		deployment.ReleaseName = n.deploymentManager.GenerateReleaseName(clusterGroup)
	}

	if !n.deploymentManager.IsReleaseNameAvailable(clusterGroup, deployment.ReleaseName, deployment.Namespace) {
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

	if n.returnOperationErrorsIfAny(c, targetClusterStatus, deployment.ReleaseName) {
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
