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

package action

import (
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/pkg/cluster/acsk"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// CreateACSKNodePoolAction describes the properties of an Alibaba cluster creation
type CreateACSKNodePoolAction struct {
	log       logrus.FieldLogger
	nodePools []*model.ACSKNodePoolModel
	context   *ACKContext
	region    string
}

// NewCreateACSKNodePoolAction creates a new CreateACSKNodePoolAction
func NewCreateACSKNodePoolAction(log logrus.FieldLogger, nodepools []*model.ACSKNodePoolModel, clusterContext *ACKContext, region string) *CreateACSKNodePoolAction {
	return &CreateACSKNodePoolAction{
		log:       log,
		nodePools: nodepools,
		context:   clusterContext,
		region:    region,
	}
}

// GetName returns the name of this CreateACSKNodePoolAction
func (a *CreateACSKNodePoolAction) GetName() string {
	return "CreateACSKNodePoolAction"
}

// ExecuteAction executes this CreateACSKNodePoolAction
func (a *CreateACSKNodePoolAction) ExecuteAction(input interface{}) (interface{}, error) {
	cluster, ok := input.(*acsk.AlibabaDescribeClusterResponse)
	if !ok {
		return nil, errors.New("invalid input")
	}

	if len(a.nodePools) == 0 {
		r, err := getClusterDetails(a.context.ClusterID, a.context.CSClient)
		if err != nil {
			return nil, emperror.With(err, "cluster", cluster.Name)
		}

		return r, nil
	}
	a.log.Infoln("EXECUTE CreateACSKNodePoolAction, cluster name", cluster.Name)

	errChan := make(chan error, len(a.nodePools))
	instanceIdsChan := make(chan []string, len(a.nodePools))
	defer close(errChan)
	defer close(instanceIdsChan)

	for _, nodePool := range a.nodePools {
		// TODO: run node pool creation in parallel once Alibaba ESS API permits running multiple CreateScalingGroupRequest in parallel
		// TODO: Currently running multiple CreateScalingGroupRequest in parallel may fail with throttling error
		createNodePool(a.log, nodePool, a.context.ESSClient, cluster, instanceIdsChan, errChan)
	}

	caughtErrors := emperror.NewMultiErrorBuilder()

	var instanceIds []string
	var err error
	for i := 0; i < len(a.nodePools); i++ {
		err = <-errChan
		ids := <-instanceIdsChan
		if err != nil {
			caughtErrors.Add(err)
		} else {
			instanceIds = append(instanceIds, ids...)
		}
	}
	err = caughtErrors.ErrOrNil()
	if err != nil {
		return nil, pkgErrors.NewMultiErrorWithFormatter(err)
	}

	return attachInstancesToCluster(a.log, cluster.ClusterID, instanceIds, a.context.CSClient)
}

// UndoAction rolls back this CreateACSKNodePoolAction
func (a *CreateACSKNodePoolAction) UndoAction() (err error) {
	a.log.Info("EXECUTE UNDO CreateACSKNodePoolAction")
	return deleteNodePools(a.log, a.nodePools, a.context.ESSClient, a.region)
}
