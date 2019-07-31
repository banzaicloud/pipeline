// Copyright © 2018 Banzai Cloud
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

package api

import (
	"net/http"
	"strconv"

	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	anchore "github.com/banzaicloud/pipeline/internal/security"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/gin-gonic/gin"
)

// DeleteClusterResponse describes Pipeline's DeleteCluster API response
type DeleteClusterResponse struct {
	Status     int    `json:"status"`
	Name       string `json:"name"`
	Message    string `json:"message"`
	ResourceID uint   `json:"id"`
}

// DeleteCluster deletes a K8S cluster from the cloud
func (a *ClusterAPI) DeleteCluster(c *gin.Context) {
	commonCluster, ok := a.clusterGetter.GetClusterFromRequest(c)
	if ok != true {
		return
	}

	force, _ := strconv.ParseBool(c.DefaultQuery("force", "false"))

	// DeleteCluster deletes the underlying model, so we get this data here
	clusterID, clusterName := commonCluster.GetID(), commonCluster.GetName()

	ctx := ginutils.Context(c.Request.Context(), c)

	// delete cluster from cluster group
	err := a.clusterGroupManager.RemoveClusterFromGroup(ctx, clusterID)
	if err != nil {
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Error:   err.Error(),
		})
		return
	}

	switch {
	case commonCluster.GetDistribution() == pkgCluster.PKE && commonCluster.GetCloud() == pkgCluster.Azure:
		if err := a.clusterDeleters.PKEOnAzure.DeleteByID(ctx, commonCluster.GetID(), force); err != nil {
			pkgCommon.ErrorResponseWithStatus(c, http.StatusInternalServerError, err)
			return
		}
	default:
		_ = a.clusterManager.DeleteCluster(ctx, commonCluster, force)
	}

	if anchore.AnchoreEnabled && commonCluster.GetSecurityScan() {
		anchore.RemoveAnchoreUser(commonCluster.GetOrganizationId(), commonCluster.GetUID())
	}

	c.JSON(http.StatusAccepted, DeleteClusterResponse{
		Status:     http.StatusAccepted,
		Name:       clusterName,
		ResourceID: clusterID,
	})
}
