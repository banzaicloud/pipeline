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

package common

import (
	"context"
	"fmt"
	"net/http"

	"emperror.dev/emperror"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/cluster"
)

type ClusterGetter interface {
	GetClusterFromRequest(c *gin.Context) (cluster.CommonCluster, bool)
}

type clusterGetter struct {
	manager      *cluster.Manager
	logger       logrus.FieldLogger
	errorHandler emperror.Handler
}

// NewClusterGetter returns a new ClusterGetter instance which returns a cluster from an API request.
func NewClusterGetter(
	manager *cluster.Manager,
	logger logrus.FieldLogger,
	errorHandler emperror.Handler,
) ClusterGetter {
	return &clusterGetter{
		manager:      manager,
		logger:       logger,
		errorHandler: errorHandler,
	}
}

// GetClusterFromRequest returns a cluster from an API request.
func (g *clusterGetter) GetClusterFromRequest(c *gin.Context) (cluster.CommonCluster, bool) {
	var cl cluster.CommonCluster
	var err error

	logger := correlationid.Logger(g.logger, c)

	ctx := ginutils.Context(context.Background(), c)

	organizationID := auth.GetCurrentOrganization(c.Request).ID

	logger = logger.WithField("organization", organizationID)

	switch field := c.DefaultQuery("field", "id"); field {
	case "id":
		clusterID, ok := ginutils.UintParam(c, "id")
		if !ok {
			logger.Debug("invalid ID parameter")

			return nil, false
		}

		logger = logger.WithField("cluster", clusterID)

		cl, err = g.manager.GetClusterByID(ctx, organizationID, clusterID)
	case "name":
		clusterName := c.Param("id")

		logger = logger.WithField("cluster", clusterName)

		cl, err = g.manager.GetClusterByName(ctx, organizationID, clusterName)
	default:
		err = fmt.Errorf("field=%s is not supported", field)
	}

	if isNotFound(err) {
		logger.Debug("cluster not found")

		c.JSON(http.StatusNotFound, pkgCommon.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "cluster not found",
			Error:   err.Error(),
		})

		return nil, false
	} else if err != nil {
		g.errorHandler.Handle(err)

		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "error parsing request",
			Error:   err.Error(),
		})

		return nil, false
	}

	return cl, true
}
