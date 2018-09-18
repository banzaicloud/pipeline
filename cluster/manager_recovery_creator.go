// Copyright © 2018 Banzai Cloud
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

type recoveryCreator struct {
	cluster CommonCluster
}

// NewRecoveryClusterCreator returns a new cluster creator instance.
func NewRecoveryClusterCreator(cluster CommonCluster) *recoveryCreator {
	return &recoveryCreator{
		cluster: cluster,
	}
}

// Validate implements the clusterCreator interface.
func (c *recoveryCreator) Validate(ctx context.Context) error {
	// We are past validation
	return nil
}

// Prepare implements the clusterCreator interface.
func (c *recoveryCreator) Prepare(ctx context.Context) (CommonCluster, error) {
	// We are past preparation
	return c.cluster, nil
}

// Create implements the clusterCreator interface.
func (c *recoveryCreator) Create(ctx context.Context) error {
	return c.cluster.CreateCluster()
}
