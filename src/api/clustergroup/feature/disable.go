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

	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	"github.com/banzaicloud/pipeline/src/auth"
)

// @Summary Disable Feature of Cluster Group
// @Description disable feature on all members of a cluster group
// @Tags clustergroup features
// @Accept json
// @Produce json
// @Param orgid path uint true "Organization ID"
// @Param clusterGroupId path uint true "Cluster Group ID"
// @Param featureName path string true "Name of the feature"
// @Success 200 {string} no content
// @Failure 400 {object} common.ErrorResponse Feature Not Found
// @Router /api/v1/orgs/{orgid}/clustergroups/{clusterGroupId}/features/{featureName} [delete]
// @Security bearerAuth
func (n *API) Disable(c *gin.Context) {
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

	err = n.clusterGroupManager.DisableFeature(featureName, clusterGroup)
	if err != nil {
		n.errorHandler.Handle(c, err)
		return
	}

	err = n.clusterGroupManager.ReconcileFeature(*clusterGroup, featureName)
	if err != nil {
		n.errorHandler.Handle(c, err)
		return
	}

	c.Status(http.StatusOK)
}
