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
	"github.com/banzaicloud/pipeline/pkg/cluster/ack"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// CreateACKNodePoolAction describes the properties of an Alibaba cluster creation
type CreateACKNodePoolAction struct {
	log       logrus.FieldLogger
	nodePools []*model.ACKNodePoolModel
	context   *ACKContext
	region    string
}

// NewCreateACKNodePoolAction creates a new CreateACKNodePoolAction
func NewCreateACKNodePoolAction(log logrus.FieldLogger, nodepools []*model.ACKNodePoolModel, clusterContext *ACKContext, region string) *CreateACKNodePoolAction {
	return &CreateACKNodePoolAction{
		log:       log,
		nodePools: nodepools,
		context:   clusterContext,
		region:    region,
	}
}

// GetName returns the name of this CreateACKNodePoolAction
func (a *CreateACKNodePoolAction) GetName() string {
	return "CreateACKNodePoolAction"
}

// ExecuteAction executes this CreateACKNodePoolAction
func (a *CreateACKNodePoolAction) ExecuteAction(input interface{}) (interface{}, error) {
	cluster, ok := input.(*ack.AlibabaDescribeClusterResponse)
	if !ok {
		return nil, errors.New("invalid input")
	}

	if len(a.nodePools) == 0 {
		r, err := GetClusterDetails(a.context.CSClient, a.context.ClusterID)
		if err != nil {
			return nil, emperror.With(err, "cluster", cluster.Name)
		}

		return r, nil
	}
	a.log.Infoln("EXECUTE CreateACKNodePoolAction, cluster name", cluster.Name)

	errChan := make(chan error, len(a.nodePools))
	instanceIdsChan := make(chan []string, len(a.nodePools))
	defer close(errChan)
	defer close(instanceIdsChan)

	for _, nodePool := range a.nodePools {
		go createNodePool(a.log, nodePool, a.context.ESSClient, cluster, instanceIdsChan, errChan)
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

// UndoAction rolls back this CreateACKNodePoolAction
func (a *CreateACKNodePoolAction) UndoAction() (err error) {
	a.log.Info("EXECUTE UNDO CreateACKNodePoolAction")
	return deleteNodePools(a.log, a.nodePools, a.context.ESSClient, a.region)
}
