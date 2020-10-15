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

package clusteradapter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/cadence/mocks"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/clusterworkflow"
)

func TestNodePoolManager_CreateNodePool(t *testing.T) {
	ctx := context.Background()
	const clusterID = uint(1)
	const nodePoolName = "pool0"

	rawNewNodePool := cluster.NewRawNodePool{
		"name": nodePoolName,
	}

	client := new(mocks.Client)
	client.On(
		"StartWorkflow",
		ctx,
		mock.Anything,
		clusterworkflow.CreateNodePoolWorkflowName,
		clusterworkflow.CreateNodePoolWorkflowInput{
			ClusterID:   clusterID,
			UserID:      1,
			RawNodePool: rawNewNodePool,
		},
	).Return(nil, nil)

	manager := NewNodePoolManager(client, func(ctx context.Context) uint { return 1 })

	err := manager.CreateNodePool(ctx, clusterID, rawNewNodePool)
	require.NoError(t, err)

	client.AssertExpectations(t)
}
