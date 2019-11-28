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

package clustergroup

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/banzaicloud/pipeline/auth"
	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
)

// @Summary Delete Cluster Group
// @Description delete a cluster group, disable all enabled features, delete related deployments
// @Tags clustergroups
// @Accept json
// @Produce json
// @Param orgid path int true "Organization ID"
// @Param clusterGroupId path int true "Cluster Group ID"
// @Success 200 {string} no content
// @Failure 400 {object} common.ErrorResponse
// @Router /api/v1/orgs/{orgid}/clustergroups/{clusterGroupId} [delete]
// @Security bearerAuth
func (n *API) Delete(c *gin.Context) {
	ctx := ginutils.Context(context.Background(), c)

	clusterGroupID, ok := ginutils.UintParam(c, "id")
	if !ok {
		return
	}

	orgID := auth.GetCurrentOrganization(c.Request).ID
	err := n.clusterGroupManager.DeleteClusterGroupByID(ctx, orgID, clusterGroupID)
	if err != nil {
		n.errorHandler.Handle(c, err)
		return
	}

	c.Status(http.StatusOK)
}
