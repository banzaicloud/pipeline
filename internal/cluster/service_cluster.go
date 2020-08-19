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

package cluster

import (
	"context"
)

// UpdateCluster updates the specified cluster.
func (s service) UpdateCluster(ctx context.Context, clusterIdentifier Identifier, update ClusterUpdate) error {
	var (
		c   Cluster
		err error
	)

	if clusterIdentifier.ClusterName != "" {
		c, err = s.clusters.GetClusterByName(ctx, clusterIdentifier.OrganizationID, clusterIdentifier.ClusterName)
	} else {
		c, err = s.clusters.GetCluster(ctx, clusterIdentifier.ClusterID)
	}

	if err != nil {
		return err
	}

	if err := s.clusters.SetStatus(ctx, c.ID, Updating, UpdatingMessage); err != nil {
		return err
	}

	service, err := s.getDistributionService(c)
	if err != nil {
		return err
	}

	return service.UpdateCluster(ctx, clusterIdentifier, update)
}
