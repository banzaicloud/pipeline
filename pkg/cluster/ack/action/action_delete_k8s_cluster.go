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
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ess"
	"github.com/banzaicloud/pipeline/model"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
)

// ACKClusterDeleteContext describes the fields used across ACK cluster delete operation
type ACKClusterDeleteContext struct {
	ACKContext
	RegionId    string
	ClusterName string
	NodePools   []*model.ACKNodePoolModel
}

// NewACKClusterDeletionContext creates a new ACKClusterDeleteContext
func NewACKClusterDeletionContext(csClient *cs.Client,
	ecsClient *ecs.Client, essClient *ess.Client, clusterID string, nodePools []*model.ACKNodePoolModel, clusterName, regionID string) *ACKClusterDeleteContext {
	return &ACKClusterDeleteContext{
		ACKContext: ACKContext{
			CSClient:  csClient,
			ECSClient: ecsClient,
			ESSClient: essClient,
			ClusterID: clusterID,
		},
		RegionId:    regionID,
		ClusterName: clusterName,
		NodePools:   nodePools,
	}
}

// DeleteACKClusterAction describes the properties of an Alibaba cluster deletion
type DeleteACKClusterAction struct {
	context *ACKClusterDeleteContext
	log     logrus.FieldLogger
}

// NewCreateACKClusterAction creates a new CreateACKClusterAction
func NewDeleteACKClusterAction(log logrus.FieldLogger, deletionContext *ACKClusterDeleteContext) *DeleteACKClusterAction {
	return &DeleteACKClusterAction{
		context: deletionContext,
		log:     log,
	}
}

// GetName returns the name of this DeleteACKClusterAction
func (a *DeleteACKClusterAction) GetName() string {
	return "DeleteACKClusterAction"
}

// ExecuteAction executes this DeleteACKClusterAction
func (a *DeleteACKClusterAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	a.log.Info("EXECUTE DeleteClusterAction")
	return nil, emperror.With(deleteCluster(a.log, a.context.ClusterID, a.context.CSClient), "cluster", a.context.ClusterName)
}
