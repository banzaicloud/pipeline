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
	"emperror.dev/emperror"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/api/clustergroup/common"
	cgroup "github.com/banzaicloud/pipeline/internal/clustergroup"
	"github.com/banzaicloud/pipeline/internal/clustergroup/deployment"
)

const (
	IDParamName = "name"
)

type API struct {
	clusterGroupManager *cgroup.Manager
	deploymentManager   *deployment.CGDeploymentManager
	logger              logrus.FieldLogger
	errorHandler        common.ErrorHandler
}

func NewAPI(
	clusterGroupManager *cgroup.Manager,
	deploymentManager *deployment.CGDeploymentManager,
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

// AddRoutes adds cluster group deployments related API routes
func (a *API) AddRoutes(group *gin.RouterGroup) {
	group.POST("", a.Create)
	group.GET("", a.List)
	item := group.Group("/:" + IDParamName)
	{
		item.GET("", a.Get)
		item.PUT("", a.Upgrade)
		item.DELETE("", a.Delete)
		item.PUT("/sync", a.Sync)
	}
}
