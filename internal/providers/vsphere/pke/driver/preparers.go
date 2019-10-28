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

package driver

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/providers/vsphere/pke"
)

// NodePoolsPreparer implements []NodePool preparation
type NodePoolsPreparer struct {
	logger       logrus.FieldLogger
	namespace    string
	dataProvider nodePoolsDataProvider
}

type nodePoolsDataProvider interface {
	getExistingNodePools(ctx context.Context) ([]pke.NodePool, error)
	getExistingNodePoolByName(ctx context.Context, nodePoolName string) (pke.NodePool, error)
}

func (p NodePoolsPreparer) getNodePoolPreparer(i int) NodePoolPreparer {
	return NodePoolPreparer{
		logger:       p.logger,
		namespace:    fmt.Sprintf("%s[%d]", p.namespace, i),
		dataProvider: p.dataProvider,
	}
}

// Prepare validates and provides defaults for a set of NodePools
func (p NodePoolsPreparer) Prepare(ctx context.Context, nodePools []NodePool) error {
	// check incoming node pool list item uniqueness
	{
		names := make(map[string]bool)
		for _, np := range nodePools {
			if names[np.Name] {
				return validationErrorf("multiple node pools named %q", np.Name)
			}
			names[np.Name] = true
		}
	}

	for i := range nodePools {
		np := &nodePools[i]

		if err := p.getNodePoolPreparer(i).Prepare(ctx, np); err != nil {
			return errors.WrapIf(err, "failed to prepare node pool")
		}
	}

	return nil
}

// NodePoolPreparer implements NodePool preparation
type NodePoolPreparer struct {
	logger       logrus.FieldLogger
	namespace    string
	dataProvider interface {
		getExistingNodePoolByName(ctx context.Context, nodePoolName string) (pke.NodePool, error)
	}
}

// Prepare validates and provides defaults for NodePool fields
func (p NodePoolPreparer) Prepare(ctx context.Context, nodePool *NodePool) error {
	if nodePool == nil {
		return nil
	}

	if nodePool.Name == "" {
		return validationErrorf("%s.Name must be specified", p.namespace)
	}

	np, err := p.dataProvider.getExistingNodePoolByName(ctx, nodePool.Name)
	if pke.IsNotFound(err) {
		return p.prepareNewNodePool(ctx, nodePool)
	} else if err != nil {
		return errors.WrapIf(err, "failed to get node pool by name")
	}

	return p.prepareExistingNodePool(ctx, nodePool, np)
}

func (p NodePoolPreparer) getLogger() logrus.FieldLogger {
	return p.logger
}

func (p NodePoolPreparer) getNamespace() string {
	return p.namespace
}

func (p NodePoolPreparer) prepareNewNodePool(ctx context.Context, nodePool *NodePool) error {
	if len(nodePool.Roles) == 0 {
		nodePool.Roles = []string{"worker"}
		p.logger.Debugf("%s.Roles not specified, defaulting to %v", p.namespace, nodePool.Roles)
	}

	return nil
}

func (p NodePoolPreparer) prepareExistingNodePool(ctx context.Context, nodePool *NodePool, existing pke.NodePool) error {
	nodePool.CreatedBy = existing.CreatedBy
	nodePool.Roles = existing.Roles
	return nil
}

type validationError struct {
	msg string
}

func validationErrorf(msg string, args ...interface{}) validationError {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	return validationError{
		msg: msg,
	}
}

func (e validationError) Error() string {
	return e.msg
}

func (e validationError) InputValidationError() bool {
	return true
}
