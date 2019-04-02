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
	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
	"go.uber.org/cadence/client"
)

type tokenGenerator interface {
	GenerateClusterToken(orgID uint, clusterID uint) (string, string, error)
}

type LeaderInfo struct {
	Hostname string
	IP       string
}

type LeaderRepository interface {
	GetLeader(organizationID, clusterID uint) (*LeaderInfo, error)
	SetLeader(organizationID, clusterID uint, leaderInfo LeaderInfo) error
	DeleteLeader(organizationID, clusterID uint) error
}

type API struct {
	clusterGetter   common.ClusterGetter
	errorHandler    emperror.Handler
	tokenGenerator  tokenGenerator
	externalBaseURL string

	workflowClient   client.Client
	leaderRepository LeaderRepository
}

func NewAPI(
	clusterGetter common.ClusterGetter,
	errorHandler emperror.Handler,
	tokenGenerator tokenGenerator,
	externalBaseURL string,
	workflowClient client.Client,
	leaderRepository LeaderRepository,
) *API {
	return &API{
		clusterGetter:    clusterGetter,
		errorHandler:     errorHandler,
		tokenGenerator:   tokenGenerator,
		externalBaseURL:  externalBaseURL,
		workflowClient:   workflowClient,
		leaderRepository: leaderRepository,
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
	r.GET("ready", a.GetReady)
	r.POST("ready", a.PostReady)
	r.POST("leader", a.PostLeaderElection)
	r.GET("leader", a.GetLeaderElection)
	r.DELETE("leader", a.DeleteLeaderElection)
}
