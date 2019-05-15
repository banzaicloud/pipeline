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

// @Summary List Cluster Groups of an Organization
// @Description retrieve list of cluster groups of an organization
// @Tags clustergroups
// @Accept json
// @Produce json
// @Param orgid path int true "Organization ID"
// @Success 200 {array} api.ClusterGroup
// @Failure 400 {object} common.ErrorResponse Cluster Group Not Found
// @Router /api/v1/orgs/{orgid}/clustergroups [get]
// @Security bearerAuth
func (a *API) List(c *gin.Context) {
	ctx := ginutils.Context(context.Background(), c)

	orgID := auth.GetCurrentOrganization(c.Request).ID
	clusterGroups, err := a.clusterGroupManager.GetAllClusterGroups(ctx, orgID)
	if err != nil {
		a.errorHandler.Handle(c, err)
		return
	}

	c.JSON(http.StatusOK, clusterGroups)
}
