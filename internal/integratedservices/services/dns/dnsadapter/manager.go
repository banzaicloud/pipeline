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

package dnsadapter

import (
	"context"

	"emperror.dev/errors"
)

// ClusterPropertyGetter can be used to get a cluster's properties
type ClusterPropertyGetter struct {
	clusterGetter CommonClusterGetter
}

// NewClusterPropertyGetter returns a new ClusterPropertyGetter instance
func NewClusterPropertyGetter(clusterGetter CommonClusterGetter) ClusterPropertyGetter {
	return ClusterPropertyGetter{
		clusterGetter: clusterGetter,
	}
}

// GetClusterOrgID returns the specified cluster's organization ID
func (g ClusterPropertyGetter) GetClusterOrgID(ctx context.Context, clusterID uint) (uint, error) {
	cluster, err := g.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return 0, errors.WrapIf(err, "failed to get cluster")
	}

	return cluster.GetOrganizationId(), nil
}

// GetClusterUID returns the specified cluster's UID
func (g ClusterPropertyGetter) GetClusterUID(ctx context.Context, clusterID uint) (string, error) {
	cluster, err := g.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return "", errors.WrapIf(err, "failed to get cluster")
	}

	return cluster.GetUID(), nil
}
