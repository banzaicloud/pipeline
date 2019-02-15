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

package pke

import (
	"github.com/banzaicloud/pipeline/api/common"
	"github.com/banzaicloud/pipeline/cluster"
	pkgAuth "github.com/banzaicloud/pipeline/pkg/auth"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
)

type tokenGenerator interface {
	GenerateClusterToken(orgID pkgAuth.OrganizationID, clusterID pkgCluster.ClusterID) (string, string, error)
}

type API struct {
	clusterGetter   common.ClusterGetter
	errorHandler    emperror.Handler
	tokenGenerator  tokenGenerator
	externalBaseURL string
}

func NewAPI(clusterGetter common.ClusterGetter, errorHandler emperror.Handler, tokenGenerator tokenGenerator, externalBaseURL string) *API {
	return &API{
		clusterGetter:   clusterGetter,
		errorHandler:    errorHandler,
		tokenGenerator:  tokenGenerator,
		externalBaseURL: externalBaseURL,
	}
}

func (a *API) getCluster(c *gin.Context) (cluster.CommonCluster, logrus.FieldLogger, bool) {
	cluster, ok := a.clusterGetter.GetClusterFromRequest(c)
	if !ok {
		return cluster, nil, ok
	}
	log := logrus.WithField("cluster", cluster.GetName()).WithField("organization", cluster.GetOrganizationId())
	return cluster, log, ok
}

func (a *API) RegisterRoutes(r gin.IRouter) {
	r.GET("commands", a.ListCommands)
	r.POST("ready", a.PostReady)
}
