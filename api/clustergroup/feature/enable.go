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
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/clustergroup"
	cgroupIAPI "github.com/banzaicloud/pipeline/internal/clustergroup/api"
	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
)

// @Summary Enable Feature of Cluster Group
// @Description enable feature on all members of a cluster group
// @Tags clustergroup features
// @Accept json
// @Produce json
// @Param orgid path uint true "Organization ID"
// @Param clusterGroupId path uint true "Cluster Group ID"
// @Param featureName path string true "Name of the feature"
// @Param fprop body api.FeatureRequest true "Feature properties"
// @Success 202 {string} no content
// @Failure 400 {object} common.ErrorResponse Feature Not Found
// @Failure 404 {object} common.ErrorResponse
// @Router /api/v1/orgs/{orgid}/clustergroups/{clusterGroupId}/features/{featureName} [post]
// @Security bearerAuth
func (n *API) Enable(c *gin.Context) {
	ctx := ginutils.Context(c)

	var req cgroupIAPI.FeatureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		n.errorHandler.Handle(c, c.Error(err).SetType(gin.ErrorTypeBind))
		return
	}

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

	err = n.clusterGroupManager.EnableFeature(featureName, clusterGroup, req)
	if err != nil {
		n.errorHandler.Handle(c, err)
		return
	}

	err = n.clusterGroupManager.ReconcileFeature(*clusterGroup, featureName)
	if err != nil {
		if !clustergroup.IsFeatureReconcileError(err) {
			n.errorHandler.Handle(c, err)
			return
		}
		// don't return with error status in case of a FeatureReconcileError since the feature is created
		n.logger.Error(err)
		c.JSON(http.StatusCreated, &pkgCommon.ErrorResponse{
			Code:    http.StatusCreated,
			Message: "Failed to reconcile feature(s)",
			Error:   err.Error(),
		})
	}

	c.Status(http.StatusCreated)
}
