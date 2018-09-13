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
	"context"
	"net/http"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/config"
	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// UpdateClusterResponse describes Pipeline's UpdateCluster API response
type UpdateClusterResponse struct {
	Status int `json:"status"`
}

// UpdateCluster updates a K8S cluster in the cloud (e.g. autoscale)
func UpdateCluster(c *gin.Context) {

	// bind request body to UpdateClusterRequest struct
	var updateRequest *pkgCluster.UpdateClusterRequest
	if err := c.BindJSON(&updateRequest); err != nil {
		log.Errorf("Error parsing request: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}
	commonCluster, ok := getClusterFromRequest(c)
	if ok != true {
		return
	}

	// TODO: move these to a struct and create them only once upon application init
	clusters := intCluster.NewClusters(config.DB())
	secretValidator := providers.NewSecretValidator(secret.Store)
	clusterManager := cluster.NewManager(clusters, secretValidator, log, errorHandler)

	updateCtx := cluster.UpdateContext{
		OrganizationID: auth.GetCurrentOrganization(c.Request).ID,
		UserID:         auth.GetCurrentUser(c.Request).ID,
		ClusterID:      commonCluster.GetID(),
	}

	updater := cluster.NewCommonClusterUpdater(updateRequest, commonCluster, updateCtx.UserID)

	ctx := ginutils.Context(context.Background(), c)

	err := clusterManager.UpdateCluster(ctx, updateCtx, updater)
	if err != nil {
		if isInvalid(err) {
			c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: errors.Cause(err).Error(),
			})

			return
		} else if isPreconditionFailed(err) {
			c.JSON(http.StatusPreconditionFailed, pkgCommon.ErrorResponse{
				Code:    http.StatusPreconditionFailed,
				Message: errors.Cause(err).Error(),
			})

			return
		} else {
			errorHandler.Handle(err)

			c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "cluster update failed",
			})

			return
		}
	}

	c.JSON(http.StatusAccepted, UpdateClusterResponse{
		Status: http.StatusAccepted,
	})
}
