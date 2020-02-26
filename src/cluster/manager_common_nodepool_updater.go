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
	"fmt"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/pkg/cluster"
)

type commonNodepoolUpdater struct {
	request *cluster.UpdateNodePoolsRequest
	cluster CommonCluster
	userID  uint
}

type commonNodepoolUpdateValidationError struct {
	msg string

	invalidRequest     bool
	preconditionFailed bool
}

func (e *commonNodepoolUpdateValidationError) Error() string {
	return e.msg
}

func (e *commonNodepoolUpdateValidationError) IsInvalid() bool {
	return e.invalidRequest
}

func (e *commonNodepoolUpdateValidationError) IsPreconditionFailed() bool {
	return e.preconditionFailed
}

// NewCommonNodepoolUpdater returns a new cluster creator instance.
func NewCommonNodepoolUpdater(request *cluster.UpdateNodePoolsRequest, cluster CommonCluster, userID uint) *commonNodepoolUpdater {
	return &commonNodepoolUpdater{
		request: request,
		cluster: cluster,
		userID:  userID,
	}
}

// Validate implements the clusterUpdater interface.
func (c *commonNodepoolUpdater) Validate(ctx context.Context) error {
	status, err := c.cluster.GetStatus()
	if err != nil {
		return errors.WrapIf(err, "could not get cluster status")
	}

	if status.Status != cluster.Running && status.Status != cluster.Warning {
		return errors.WithDetails(
			&commonNodepoolUpdateValidationError{
				msg:                fmt.Sprintf("cluster is not in %s or %s state yet", cluster.Running, cluster.Warning),
				preconditionFailed: true,
			},
			"status", status.Status,
		)
	}

	// check node pools
	for poolName := range c.request.NodePools {
		if !c.cluster.NodePoolExists(poolName) {
			return errors.WithDetails(
				&commonNodepoolUpdateValidationError{
					msg:            fmt.Sprintf("Unable to find node pool with name: %s", poolName),
					invalidRequest: true,
				},
				"status", status.Status,
			)
		}
	}

	return nil
}

// Prepare implements the clusterUpdater interface.
func (c *commonNodepoolUpdater) Prepare(ctx context.Context) (CommonCluster, error) {
	if err := c.cluster.SetStatus(cluster.Updating, cluster.UpdatingMessage); err != nil {
		return nil, err
	}
	return c.cluster, c.cluster.Persist()
}

// Update implements the clusterUpdater interface.
func (c *commonNodepoolUpdater) Update(ctx context.Context) error {
	return c.cluster.UpdateNodePools(c.request, c.userID)
}
