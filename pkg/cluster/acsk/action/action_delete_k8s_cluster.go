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
	"github.com/sirupsen/logrus"
)

// ACSKClusterDeleteContext describes the fields used across ACSK cluster delete operation
type ACSKClusterDeleteContext struct {
	ACSKClusterContext
	RegionId  string
	NodePools []*model.ACSKNodePoolModel
}

// NewACSKClusterDeletionContext creates a new ACSKClusterDeleteContext
func NewACSKClusterDeletionContext(csClient *cs.Client,
	ecsClient *ecs.Client, essClient *ess.Client, clusterID string, nodePools []*model.ACSKNodePoolModel, regionID string) *ACSKClusterDeleteContext {
	return &ACSKClusterDeleteContext{
		ACSKClusterContext: ACSKClusterContext{
			CSClient:  csClient,
			ECSClient: ecsClient,
			ESSClient: essClient,
			ClusterID: clusterID,
		},
		RegionId:  regionID,
		NodePools: nodePools,
	}
}

// DeleteACSKClusterAction describes the properties of an Alibaba cluster deletion
type DeleteACSKClusterAction struct {
	context *ACSKClusterDeleteContext
	log     logrus.FieldLogger
}

// NewCreateACSKClusterAction creates a new CreateACSKClusterAction
func NewDeleteACSKClusterAction(log logrus.FieldLogger, deletionContext *ACSKClusterDeleteContext) *DeleteACSKClusterAction {
	return &DeleteACSKClusterAction{
		context: deletionContext,
		log:     log,
	}
}

// GetName returns the name of this DeleteACSKClusterAction
func (a *DeleteACSKClusterAction) GetName() string {
	return "DeleteACSKClusterAction"
}

// ExecuteAction executes this DeleteACSKClusterAction
func (a *DeleteACSKClusterAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	a.log.Info("EXECUTE DeleteClusterAction")
	return nil, deleteCluster(a.context.ClusterID, a.context.CSClient)
}
