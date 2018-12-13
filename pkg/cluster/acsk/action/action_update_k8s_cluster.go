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
	"encoding/json"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cs"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/pkg/cluster/acsk"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// UpdateACSKClusterAction describes the fields used across ACSK cluster update operation
type UpdateACSKClusterAction struct {
	log       logrus.FieldLogger
	nodePools []*model.ACSKNodePoolModel
	context   *ACSKClusterContext
}

// NewUpdateACSKClusterAction creates a new UpdateACSKClusterAction
func NewUpdateACSKClusterAction(log logrus.FieldLogger, nodepools []*model.ACSKNodePoolModel, clusterContext *ACSKClusterContext) *UpdateACSKClusterAction {
	return &UpdateACSKClusterAction{
		log:       log,
		nodePools: nodepools,
		context:   clusterContext,
	}
}

// GetName returns the name of this UpdateACSKClusterAction
func (a *UpdateACSKClusterAction) GetName() string {
	return "UpdateACSKClusterAction"
}

// ExecuteAction executes this UpdateACSKClusterAction
func (a *UpdateACSKClusterAction) ExecuteAction(input interface{}) (interface{}, error) {
	a.log.Infof("EXECUTE UpdateACSKClusterAction on cluster, %s", a.context.ClusterID)
	csClient := a.context.CSClient

	//setup cluster update request
	params := acsk.AlibabaScaleClusterParams{
		DisableRollback:    true,
		TimeoutMins:        60,
		WorkerInstanceType: a.nodePools[0].InstanceType,
		NumOfNodes:         a.nodePools[0].Count,
	}
	p, err := json.Marshal(&params)
	if err != nil {
		return nil, err
	}

	req := cs.CreateScaleClusterRequest()
	req.ClusterId = a.context.ClusterID
	req.SetScheme(requests.HTTPS)
	req.SetDomain(acsk.AlibabaApiDomain)
	req.SetContent(p)
	req.SetContentType(requests.Json)

	//do a cluster scale
	resp, err := csClient.ScaleCluster(req)
	if err != nil {
		a.log.Errorf("ScaleCluster error %s", err)
		return nil, err
	}

	// parse response
	var r acsk.AlibabaClusterCreateResponse
	err = json.Unmarshal(resp.GetHttpContentBytes(), &r)
	if err != nil {
		return nil, err
	}

	a.context.ClusterID = r.ClusterID

	cluster, err := waitUntilClusterCreateOrScaleComplete(a.log, r.ClusterID, csClient, false)
	if err != nil {
		return nil, errors.Wrap(err, "cluster scale failed")
	}

	return cluster, nil
}
