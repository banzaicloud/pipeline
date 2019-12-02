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

package kubernetesadapter

import (
	"context"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter"
)

// ConfigSecretGetter returns a config secret ID for a cluster.
type ConfigSecretGetter struct {
	clusters *clusteradapter.Clusters
}

// NewConfigSecretGetter returns a new ConfigSecretGetter.
func NewConfigSecretGetter(clusters *clusteradapter.Clusters) ConfigSecretGetter {
	return ConfigSecretGetter{
		clusters: clusters,
	}
}

// GetConfigSecretID returns a config secret ID for a cluster.
func (g ConfigSecretGetter) GetConfigSecretID(ctx context.Context, clusterID uint) (string, error) {
	cluster, err := g.clusters.FindOneByID(0, clusterID)
	if err != nil {
		return "", errors.WrapIf(err, "failed to get cluster")
	}

	return cluster.ConfigSecretId, nil
}
