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

package clusterdriver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/cluster"
)

func TestMakeNodePoolEndpoints_DeleteNodePool(t *testing.T) {
	ctx := context.Background()
	const clusterID = uint(1)
	const nodePoolName = "pool0"

	service := new(cluster.MockNodePoolService)
	service.On("DeleteNodePool", ctx, clusterID, nodePoolName).Return(false, nil)

	e := MakeNodePoolEndpoints(service).DeleteNodePool

	_, err := e(ctx, deleteNodePoolRequest{clusterID, nodePoolName})
	require.NoError(t, err)

	service.AssertExpectations(t)
}
