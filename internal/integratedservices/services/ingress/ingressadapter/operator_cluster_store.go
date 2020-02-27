// Copyright © 2020 Banzai Cloud
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

package ingressadapter

import (
	"context"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/ingress"
)

type OperatorClusterStore struct {
	clusterStore GenericClusterStore
}

func NewOperatorClusterStore(clusterStore GenericClusterStore) OperatorClusterStore {
	return OperatorClusterStore{
		clusterStore: clusterStore,
	}
}

type GenericClusterStore interface {
	GetCluster(ctx context.Context, id uint) (cluster.Cluster, error)
}

func (s OperatorClusterStore) Get(ctx context.Context, clusterID uint) (ingress.OperatorCluster, error) {
	c, err := s.clusterStore.GetCluster(ctx, clusterID)
	if err != nil {
		return ingress.OperatorCluster{}, err
	}

	return ingress.OperatorCluster{
		OrganizationID: c.OrganizationID,
		Cloud:          c.Cloud,
	}, nil
}
