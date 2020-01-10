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

	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/src/cluster"
)

type CommonClusterDeleterAdapter struct {
	commonClusterDeleter CommonClusterDeleter
	commonClusterGetter  CommonClusterGetter
}

func NewCommonClusterDeleterAdapter(commonClusterDeleter CommonClusterDeleter, commonClusterGetter CommonClusterGetter) CommonClusterDeleterAdapter {
	return CommonClusterDeleterAdapter{
		commonClusterDeleter: commonClusterDeleter,
		commonClusterGetter:  commonClusterGetter,
	}
}

type CommonClusterDeleter interface {
	DeleteCluster(ctx context.Context, cluster cluster.CommonCluster, force bool) error
}

type CommonClusterGetter interface {
	GetClusterByIDOnly(ctx context.Context, clusterID uint) (cluster.CommonCluster, error)
}

func (a CommonClusterDeleterAdapter) DeleteCluster(ctx context.Context, clusterID uint, options intCluster.DeleteClusterOptions) error {
	cc, err := a.commonClusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return err
	}

	return a.commonClusterDeleter.DeleteCluster(ctx, cc, options.Force)
}
