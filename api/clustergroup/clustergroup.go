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

package clustergroup

import (
	"github.com/banzaicloud/pipeline/api/clustergroup/deployment"
	pkgDep "github.com/banzaicloud/pipeline/internal/clustergroup/deployment"
	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/api/clustergroup/common"

	"github.com/banzaicloud/pipeline/api/clustergroup/feature"
	cgroup "github.com/banzaicloud/pipeline/internal/clustergroup"
)

const (
	IDParamName = "id"
)

// API implements the Cluster Group Management API actions.
type API struct {
	clusterGroupManager *cgroup.Manager
	deploymentManager   *pkgDep.CGDeploymentManager
	logger              logrus.FieldLogger
	errorHandler        common.ErrorHandler
}

func NewAPI(
	clusterGroupManager *cgroup.Manager,
	deploymentManager *pkgDep.CGDeploymentManager,
	logger logrus.FieldLogger,
	baseErrorHandler emperror.Handler,
) *API {
	return &API{
		clusterGroupManager: clusterGroupManager,
		deploymentManager:   deploymentManager,
		logger:              logger,
		errorHandler: common.ErrorHandler{
			Handler: baseErrorHandler,
		},
	}
}

// AddRoutes adds cluster group related API routes
func (a *API) AddRoutes(group *gin.RouterGroup) {
	group.GET("", a.List)
	group.POST("", a.Create)
	item := group.Group("/:" + IDParamName)
	{
		item.GET("", a.Get)
		item.PUT("", a.Update)
		item.DELETE("", a.Delete)
	}

	feature.NewAPI(a.clusterGroupManager, a.deploymentManager, a.logger, a.errorHandler.Handler).AddRoutes(item.Group("/features"))
	deployment.NewAPI(a.clusterGroupManager, a.deploymentManager, a.logger, a.errorHandler.Handler).AddRoutes(item.Group("/deployments"))
}
