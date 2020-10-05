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

package clusteradapter

import (
	"context"

	"emperror.dev/errors"
	"github.com/mitchellh/mapstructure"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke"
)

// NewPKEService returns a new PKE distribution service.
func NewPKEService(service pke.Service) cluster.Service {
	return pkeService{
		service: service,
	}
}

type pkeService struct {
	service pke.Service
}

func (s pkeService) UpdateCluster(ctx context.Context, clusterIdentifier cluster.Identifier, rawUpdate cluster.ClusterUpdate) error {
	var clusterUpdate pke.ClusterUpdate

	err := mapstructure.Decode(rawUpdate, &clusterUpdate)
	if err != nil {
		// TODO: return a service error
		return errors.Wrap(err, "failed to decode cluster update")
	}

	return s.service.UpdateCluster(ctx, clusterIdentifier.ClusterID, clusterUpdate)
}

func (s pkeService) DeleteCluster(ctx context.Context, clusterIdentifier cluster.Identifier, options cluster.DeleteClusterOptions) (deleted bool, err error) {
	panic("implement me")
}

func (s pkeService) CreateNodePool(ctx context.Context, clusterID uint, rawNodePool cluster.NewRawNodePool) error {
	panic("implement me")
}

func (s pkeService) UpdateNodePool(ctx context.Context, clusterID uint, nodePoolName string, rawNodePoolUpdate cluster.RawNodePoolUpdate) (string, error) {
	var nodePoolUpdate pke.NodePoolUpdate

	err := mapstructure.Decode(rawNodePoolUpdate, &nodePoolUpdate)
	if err != nil {
		// TODO: return a service error
		return "", errors.Wrap(err, "failed to decode node pool update")
	}

	return s.service.UpdateNodePool(ctx, clusterID, nodePoolName, nodePoolUpdate)
}

func (s pkeService) DeleteNodePool(ctx context.Context, clusterID uint, name string) (deleted bool, err error) {
	panic("implement me")
}

// ListNodePools lists node pools from a cluster.
func (s pkeService) ListNodePools(ctx context.Context, clusterID uint) (nodePoolList cluster.RawNodePoolList, err error) {
	nodePools, err := s.service.ListNodePools(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapWithDetails(err, "listing node pools through PKE service failed", "clusterID", clusterID)
	}

	nodePoolList = make([]interface{}, 0, len(nodePools))
	for _, nodePool := range nodePools {
		nodePoolList = append(nodePoolList, nodePool)
	}

	return nodePoolList, nil
}
