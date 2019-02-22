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

package pkeworkflowadapter

import (
	"github.com/banzaicloud/pipeline/pkg/cluster"
)

type tokenHandler interface {
	GenerateClusterToken(orgID uint, clusterID cluster.ClusterID) (string, string, error)
}

// ClusterManagerAdapter provides an adapter for pkeworkflow.Clusters.
type tokenGenerator struct {
	tokenHandler tokenHandler
}

// NewClusterManagerAdapter creates a new ClusterManagerAdapter.
func NewTokenGenerator(tokenHandler tokenHandler) *tokenGenerator {
	return &tokenGenerator{
		tokenHandler: tokenHandler,
	}
}

func (g *tokenGenerator) GenerateClusterToken(orgID, clusterID uint) (string, string, error) {
	return g.tokenHandler.GenerateClusterToken(orgID, cluster.ClusterID(clusterID))
}
