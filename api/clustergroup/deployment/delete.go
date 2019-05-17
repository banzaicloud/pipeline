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
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/banzaicloud/pipeline/auth"
	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
)

// @Summary Delete Cluster Group Deployment
// @Description deletes a cluster group deployment, also deleting deployments from member clusters
// @Tags clustergroup deployments
// @Accept json
// @Produce json
// @Param orgid path uint true "Organization ID"
// @Param clusterGroupId path uint true "Cluster Group ID"
// @Param deploymentName path string true "release name of a cluster group deployment"
// @Param force query boolean false "if true cluster group deployment gets deleted even if some deployments can not be deleted from each target cluster"
// @Success 202 {object} deployment.TargetClusterStatus "Multi-cluster deployment has been deleted successfully. All delete operations on all targeted clusters returned with no errors."
// @Success 207 {object} common.ErrorResponse "Partial failure, meaning that there was as least one failure on one of the target clusters"
// @Failure 400 {object} common.ErrorResponse Deployment Not Found
// @Router /api/v1/orgs/{orgid}/clustergroups/{clusterGroupId}/deployments/{deploymentName} [delete]
// @Security bearerAuth
func (n *API) Delete(c *gin.Context) {

	ctx := ginutils.Context(context.Background(), c)

	name := c.Param("name")
	// make force true by default until it's not supported by UI
	force, _ := strconv.ParseBool(c.DefaultQuery("force", "true"))
	n.logger.Infof("getting details for cluster group deployment: [%s]", name)

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

	response, err := n.deploymentManager.DeleteDeployment(clusterGroup, name, force)
	if err != nil {
		n.errorHandler.Handle(c, err)
		return
	}

	if n.returnOperationErrorsIfAny(c, response, name) {
		return
	}

	c.JSON(http.StatusAccepted, response)
	return
}
