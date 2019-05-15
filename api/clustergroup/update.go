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
	cgroupIAPI "github.com/banzaicloud/pipeline/internal/clustergroup/api"
	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
)

// @Summary Update Cluster Group
// @Description update name & member clusters for a cluster group
// @Tags clustergroups
// @Accept json
// @Produce json
// @Param orgid path int true "Organization ID"
// @Param clusterGroupId path int true "Cluster Group ID"
// @Param cgroup body api.UpdateRequest true "Update Cluster Group Request"
// @Success 202 {object} api.UpdateResponse
// @Failure 400 {object} common.ErrorResponse Cluster Group Not Found
// @Failure 404 {object} common.ErrorResponse
// @Router /api/v1/orgs/{orgid}/clustergroups/{clusterGroupId} [put]
// @Security bearerAuth
func (n *API) Update(c *gin.Context) {
	ctx := ginutils.Context(context.Background(), c)
	clusterGroupId, ok := ginutils.UintParam(c, "id")
	if !ok {
		return
	}

	var req cgroupIAPI.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		n.errorHandler.Handle(c, c.Error(err).SetType(gin.ErrorTypeBind))
		return
	}

	orgID := auth.GetCurrentOrganization(c.Request).ID
	err := n.clusterGroupManager.UpdateClusterGroup(ctx, clusterGroupId, orgID, req.Name, req.Members)
	if err != nil {
		n.errorHandler.Handle(c, err)
		return
	}

	c.Status(http.StatusAccepted)
}
