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

	"github.com/banzaicloud/pipeline/internal/providers/vsphere/pke"
	pkgPKE "github.com/banzaicloud/pipeline/pkg/cluster/pke"
)

// NodePoolsPreparer implements []NodePool preparation
type NodePoolsPreparer struct {
	logger       Logger
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
	logger       Logger
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

	if nodePool.RAM < 4 || nodePool.RAM > 6128<<10 {
		return validationErrorf("%s.RAM must be between 4 MiB and 6128 GiB", p.namespace)
	}
	if nodePool.RAM%4 != 0 {
		return validationErrorf("%s.RAM must be multiple of 4 (MiB)", p.namespace)
	}

	if nodePool.hasRole(pkgPKE.RoleMaster) && nodePool.Size == 0 {
		p.logger.Debug("Master node pool size should be >= 0, defaulting to 1")
		nodePool.Size = 1
	}

	np, err := p.dataProvider.getExistingNodePoolByName(ctx, nodePool.Name)
	if pke.IsNotFound(err) {
		return p.prepareNewNodePool(ctx, nodePool)
	} else if err != nil {
		return errors.WrapIf(err, "failed to get node pool by name")
	}

	return p.prepareExistingNodePool(ctx, nodePool, np)
}

func (p NodePoolPreparer) getLogger() Logger {
	return p.logger
}

func (p NodePoolPreparer) getNamespace() string {
	return p.namespace
}

func (p NodePoolPreparer) prepareNewNodePool(ctx context.Context, nodePool *NodePool) error {
	if len(nodePool.Roles) == 0 {
		nodePool.Roles = []string{"worker"}
		p.logger.Debug(fmt.Sprintf("%s.Roles not specified, defaulting to %v", p.namespace, nodePool.Roles))
	}

	return nil
}

func (p NodePoolPreparer) prepareExistingNodePool(ctx context.Context, nodePool *NodePool, existing pke.NodePool) error {
	if nodePool.CreatedBy != existing.CreatedBy {
		if nodePool.CreatedBy != 0 {
			p.logMismatch("CreatedBy", existing.CreatedBy, nodePool.CreatedBy)
		}
		nodePool.CreatedBy = existing.CreatedBy
	}
	if !stringSliceSetEqual(nodePool.Roles, existing.Roles) {
		if nodePool.Roles != nil {
			p.logMismatch("Roles", existing.Roles, nodePool.Roles)
		}
		nodePool.Roles = existing.Roles
	}
	if nodePool.AdminUsername != existing.AdminUsername {
		if nodePool.AdminUsername != "" {
			p.logMismatch("AdminUsername", existing.AdminUsername, nodePool.AdminUsername)
		}
		nodePool.AdminUsername = existing.AdminUsername
	}
	if nodePool.RAM != existing.RAM {
		if nodePool.RAM > 0 {
			p.logMismatch("RAM", existing.RAM, nodePool.RAM)
		}
		nodePool.RAM = existing.RAM
	}
	if nodePool.VCPU != existing.VCPU {
		if nodePool.VCPU > 0 {
			p.logMismatch("VCPU", existing.VCPU, nodePool.VCPU)
		}
		nodePool.VCPU = existing.VCPU
	}
	if nodePool.TemplateName != existing.TemplateName {
		if nodePool.TemplateName != "" {
			p.logMismatch("TemplateName", existing.TemplateName, nodePool.TemplateName)
		}
		nodePool.TemplateName = existing.TemplateName
	}

	return nil
}

func (p NodePoolPreparer) logMismatch(fieldName string, currentValue, incomingValue interface{}) {
	p.logger.Warn(fmt.Sprintf("%s.%s does not match existing value", p.namespace, fieldName), map[string]interface{}{"current": currentValue, "incoming": incomingValue})
}

func stringSliceSetEqual(lhs, rhs []string) bool {
	lset := make(map[string]bool, len(lhs))
	for _, e := range lhs {
		lset[e] = true
	}
	if len(lhs) != len(lset) {
		return false // duplicates in lhs
	}

	rset := make(map[string]bool, len(rhs))
	for _, e := range rhs {
		rset[e] = true
	}
	if len(rhs) != len(rset) {
		return false // duplicates in rhs
	}

	if len(lset) != len(rset) {
		return false // different element counts
	}
	for e := range lset {
		if !rset[e] {
			return false // element in lhs missing from rhs
		}
	}
	return true
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
