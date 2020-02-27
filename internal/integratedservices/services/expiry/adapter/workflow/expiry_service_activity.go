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

package workflow

import (
	"context"

	"github.com/banzaicloud/pipeline/internal/cluster"
)

const ExpireActivityName = "expire-cluster-activity"

type ExpiryActivityInput struct {
	ClusterID uint
}

type ExpiryActivity struct {
	clusterDeleter clusterDeleter
}

func NewExpiryActivity(clusterDeleter clusterDeleter) ExpiryActivity {
	return ExpiryActivity{
		clusterDeleter: clusterDeleter,
	}
}

func (a ExpiryActivity) Execute(ctx context.Context, input ExpiryActivityInput) error {
	// todo revise the options argument here
	return a.clusterDeleter.DeleteCluster(ctx, input.ClusterID, cluster.DeleteClusterOptions{Force: true})
}

// clusterDeleter contract for triggering cluster deletion.
// Designed to be used by the expire integrated service
type clusterDeleter interface {
	DeleteCluster(ctx context.Context, clusterID uint, options cluster.DeleteClusterOptions) error
}
