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

	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
)

type AzurePKEClusterUpdater struct {
	paramsPreparer AzurePKEClusterUpdateParamsPreparer
}

type AzurePKEClusterUpdateParams struct {
	ClusterID uint
	NodePools []NodePool
}

func (cu AzurePKEClusterUpdater) Update(ctx context.Context, params AzurePKEClusterUpdateParams) error {
	if err := cu.paramsPreparer.Prepare(ctx, &params); err != nil {
		return emperror.Wrap(err, "params preparation failed")
	}
	return nil
}

type AzurePKEClusterUpdateParamsPreparer struct {
	logger logrus.FieldLogger
	store  pke.AzurePKEClusterStore
}

func (p AzurePKEClusterUpdateParamsPreparer) Prepare(ctx context.Context, params *AzurePKEClusterUpdateParams) error {
	if params.ClusterID == 0 {
		return validationErrorf("ClusterID cannot be 0")
	}
	exists, err := p.store.Exists(params.ClusterID)
	if err != nil {
		return emperror.Wrap(err, "failed to get cluster by ID")
	}
	if !exists {
		return validationErrorf("ClusterID must refer to an existing cluster")
	}
	nodePoolsPreparer := ClusterUpdateNodePoolsPreparer{
		clusterID: params.ClusterID,
		logger:    p.logger,
		namespace: "NodePools",
		store:     p.store,
	}
	if err := nodePoolsPreparer.Prepare(ctx, params.NodePools); err != nil {
		return emperror.Wrap(err, "failed to prepare node pools")
	}
	return nil
}

type ClusterUpdateNodePoolsPreparer struct {
	clusterID uint
	logger    logrus.FieldLogger
	namespace string
	store     pke.AzurePKEClusterStore
}

func (p ClusterUpdateNodePoolsPreparer) getNodePoolPreparer(i int) ClusterUpdateNodePoolPreparer {
	return ClusterUpdateNodePoolPreparer{
		clusterID: p.clusterID,
		logger:    p.logger,
		namespace: fmt.Sprintf("%s[%d]", p.namespace, i),
		store:     p.store,
	}
}

func (p ClusterUpdateNodePoolsPreparer) Prepare(ctx context.Context, nodePools []NodePool) error {
	for i := range nodePools {
		if err := emperror.Wrap(p.getNodePoolPreparer(i).Prepare(ctx, &nodePools[i]), "node pool preparation failed"); err != nil {
			return err
		}
	}
	// TODO: check for conflicts?
	return nil
}

type ClusterUpdateNodePoolPreparer struct {
	clusterID uint
	logger    logrus.FieldLogger
	namespace string
	store     pke.AzurePKEClusterStore
}

func (p ClusterUpdateNodePoolPreparer) getNewNodePoolPreparer() NodePoolPreparer {
	return NodePoolPreparer{
		logger:    p.logger,
		namespace: p.namespace,
	}
}

func (p ClusterUpdateNodePoolPreparer) getLogger() logrus.FieldLogger {
	return p.logger
}

func (p ClusterUpdateNodePoolPreparer) getNamespace() string {
	return p.namespace
}

func (p ClusterUpdateNodePoolPreparer) Prepare(ctx context.Context, nodePool *NodePool) error {
	if nodePool.Name == "" {
		return validationErrorf("%s.Name cannot be empty", p.namespace)
	}
	np, err := p.store.GetNodePoolByName(p.clusterID, nodePool.Name)
	if pke.IsNotFound(err) {
		return p.getNewNodePoolPreparer().Prepare(nodePool)
	} else if err != nil {
		return emperror.Wrap(err, "failed to get node pool by name")
	}

	// check existing node pool details
	if nodePool.CreatedBy != np.CreatedBy {
		if nodePool.CreatedBy != 0 {
			logMismatchOn(p, "CreatedBy", np.CreatedBy, nodePool.CreatedBy)
		}
		nodePool.CreatedBy = np.CreatedBy
	}
	if nodePool.InstanceType != np.InstanceType {
		if nodePool.InstanceType != "" {
			logMismatchOn(p, "InstanceType", np.InstanceType, nodePool.InstanceType)
		}
		nodePool.InstanceType = np.InstanceType
	}
	if stringSliceSetEqual(nodePool.Roles, np.Roles) {
		if nodePool.Roles != nil {
			logMismatchOn(p, "Roles", np.Roles, nodePool.Roles)
		}
		nodePool.Roles = np.Roles
	}
	if nodePool.Subnet.Name != np.Subnet.Name {
		if nodePool.Subnet.Name != "" {
			logMismatchOn(p, "Subnet.Name", np.Subnet.Name, nodePool.Subnet.Name)
		}
		nodePool.Subnet.Name = np.Subnet.Name
	}
	// TODO: check subnet CIDR
	if stringSliceSetEqual(nodePool.Zones, np.Zones) {
		if nodePool.Zones != nil {
			logMismatchOn(p, "Zones", np.Zones, nodePool.Zones)
		}
		nodePool.Zones = np.Zones
	}
	return nil
}

func logMismatchOn(nl interface {
	getLogger() logrus.FieldLogger
	getNamespace() string
}, fieldName string, currentValue, incomingValue interface{}) {
	logMismatch(nl.getLogger(), nl.getNamespace(), fieldName, currentValue, incomingValue)
}

func logMismatch(logger logrus.FieldLogger, namespace, fieldName string, currentValue, incomingValue interface{}) {
	logger.WithField("current", currentValue).WithField("incoming", incomingValue).Warningf("%s.%s does not match existing value", namespace, fieldName)
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
