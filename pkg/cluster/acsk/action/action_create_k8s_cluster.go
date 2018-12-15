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
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ess"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/pkg/cluster/acsk"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ACSKClusterContext describes the common fields used across ACSK cluster create/update/delete operations
type ACSKClusterContext struct {
	ClusterID string
	CSClient  *cs.Client
	ECSClient *ecs.Client
	ESSClient *ess.Client
}

// NewACSKClusterContext creates a new ACSKClusterContext
func NewACSKClusterContext(csClient *cs.Client,
	ecsClient *ecs.Client, clusterID string) *ACSKClusterContext {
	return &ACSKClusterContext{
		CSClient:  csClient,
		ECSClient: ecsClient,
		ClusterID: clusterID,
	}
}

// ACSKClusterCreateContext describes the fields used across ACSK cluster create operation
type ACSKClusterCreateContext struct {
	ACSKClusterContext
	acsk.AlibabaClusterCreateParams
	NodePools []*model.ACSKNodePoolModel
}

type AlibabaClusterFailureLogsError struct {
	clusterEventLogs []string
}

func (e AlibabaClusterFailureLogsError) Error() string {
	if len(e.clusterEventLogs) > 0 {
		return "\n" + strings.Join(e.clusterEventLogs, "\n")
	}

	return ""
}

// NewACSKClusterCreationContext creates a new ACSKClusterCreateContext
func NewACSKClusterCreationContext(
	csClient *cs.Client,
	ecsClient *ecs.Client,
	essClient *ess.Client,
	clusterCreateParams acsk.AlibabaClusterCreateParams,
	nodepools []*model.ACSKNodePoolModel,
) *ACSKClusterCreateContext {
	return &ACSKClusterCreateContext{
		ACSKClusterContext: ACSKClusterContext{
			CSClient:  csClient,
			ECSClient: ecsClient,
			ESSClient: essClient,
		},
		AlibabaClusterCreateParams: clusterCreateParams,
		NodePools:                  nodepools,
	}
}

// CreateACSKClusterAction describes the properties of an Alibaba cluster creation
type CreateACSKClusterAction struct {
	context *ACSKClusterCreateContext
	log     logrus.FieldLogger
}

// NewCreateACSKClusterAction creates a new CreateACSKClusterAction
func NewCreateACSKClusterAction(log logrus.FieldLogger, creationContext *ACSKClusterCreateContext) *CreateACSKClusterAction {
	return &CreateACSKClusterAction{
		context: creationContext,
		log:     log,
	}
}

// GetName returns the name of this CreateACSKClusterAction
func (a *CreateACSKClusterAction) GetName() string {
	return "CreateACSKClusterAction"
}

// ExecuteAction executes this CreateACSKClusterAction
func (a *CreateACSKClusterAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	a.log.Infoln("EXECUTE CreateACSKClusterAction, cluster name", a.context.Name)
	csClient := a.context.CSClient

	// setup cluster creation request
	params := a.context.AlibabaClusterCreateParams
	p, err := json.Marshal(&params)
	if err != nil {
		return nil, err
	}

	req := cs.CreateCreateClusterRequest()
	req.SetScheme(requests.HTTPS)
	req.SetDomain(acsk.AlibabaApiDomain)
	req.SetContent(p)
	req.SetContentType(requests.Json)

	// do a cluster creation
	resp, err := csClient.CreateCluster(req)
	if err != nil {
		a.log.Errorf("CreateCluster error: %s", err)
		return nil, err
	}
	if !resp.IsSuccess() || resp.GetHttpStatus() < 200 || resp.GetHttpStatus() > 299 {
		a.log.Errorf("CreateCluster error status code is: %d", resp.GetHttpStatus())
		return nil, errors.Errorf("create cluster error the returned status code is %d", resp.GetHttpStatus())
	}

	// parse response
	var r acsk.AlibabaClusterCreateResponse
	err = json.Unmarshal(resp.GetHttpContentBytes(), &r)
	if err != nil {
		return nil, err
	}

	a.log.Infof("Alibaba cluster creating with id %s", r.ClusterID)

	// We need this field to be able to implement the UndoAction for ClusterCreate
	a.context.ClusterID = r.ClusterID

	// wait for cluster created
	a.log.Info("Waiting for cluster...")
	cluster, err := waitUntilClusterCreateOrScaleComplete(a.log, r.ClusterID, csClient, true)
	if err != nil {
		return nil, emperror.WrapWith(err, "cluster create failed", "clusterName", a.context.Name)
	}

	return cluster, nil
}

// UndoAction rolls back this CreateACSKClusterAction
func (a *CreateACSKClusterAction) UndoAction() error {
	a.log.Info("EXECUTE UNDO CreateACSKClusterAction")

	_, err := waitUntilClusterCreateOrScaleComplete(a.log, a.context.ClusterID, a.context.CSClient, true)
	if err != nil {
		a.log.Warn("Error happened during waiting for cluster state to be deleted ", err)
	}
	return deleteCluster(a.context.ClusterID, a.context.CSClient)
}
