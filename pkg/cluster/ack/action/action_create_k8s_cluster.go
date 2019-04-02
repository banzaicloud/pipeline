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
	"github.com/banzaicloud/pipeline/pkg/cluster/ack"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ACKContext describes the common fields used across ACK cluster create/update/delete operations
type ACKContext struct {
	ClusterID string
	CSClient  *cs.Client
	ECSClient *ecs.Client
	ESSClient *ess.Client
}

// NewACKContext creates a new ACKContext
func NewACKContext(clusterID string, csClient *cs.Client, ecsClient *ecs.Client, essClient *ess.Client) *ACKContext {
	return &ACKContext{
		ClusterID: clusterID,
		CSClient:  csClient,
		ECSClient: ecsClient,
		ESSClient: essClient,
	}
}

// ACKClusterCreateContext describes the fields used across ACK cluster create operation
type ACKClusterCreateContext struct {
	ACKContext
	ack.AlibabaClusterCreateParams
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

// NewACKClusterCreationContext creates a new ACKClusterCreateContext
func NewACKClusterCreationContext(context ACKContext, params ack.AlibabaClusterCreateParams) *ACKClusterCreateContext {
	return &ACKClusterCreateContext{
		ACKContext:                 context,
		AlibabaClusterCreateParams: params,
	}
}

// CreateACKClusterAction describes the properties of an Alibaba cluster creation
type CreateACKClusterAction struct {
	context *ACKClusterCreateContext
	log     logrus.FieldLogger
}

// NewCreateACKClusterAction creates a new CreateACKClusterAction
func NewCreateACKClusterAction(log logrus.FieldLogger, creationContext *ACKClusterCreateContext) *CreateACKClusterAction {
	return &CreateACKClusterAction{
		context: creationContext,
		log:     log,
	}
}

// GetName returns the name of this CreateACKClusterAction
func (a *CreateACKClusterAction) GetName() string {
	return "CreateACKClusterAction"
}

// ExecuteAction executes this CreateACKClusterAction
func (a *CreateACKClusterAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	a.log.Infoln("EXECUTE CreateACKClusterAction, cluster name", a.context.Name)
	csClient := a.context.CSClient

	// setup cluster creation request
	params := a.context.AlibabaClusterCreateParams
	p, err := json.Marshal(&params)
	if err != nil {
		return nil, emperror.WrapWith(err, "could not marshal cluster create params", "cluster", a.context.Name)
	}

	req := cs.CreateCreateClusterRequest()
	req.SetScheme(requests.HTTPS)
	req.SetDomain(ack.AlibabaApiDomain)
	req.SetContent(p)
	req.SetContentType(requests.Json)

	// do a cluster creation
	resp, err := csClient.CreateCluster(req)
	if err != nil {
		return nil, emperror.WrapWith(err, "could not create cluster", "cluster", a.context.Name)
	}
	if !resp.IsSuccess() || resp.GetHttpStatus() < 200 || resp.GetHttpStatus() > 299 {
		return nil, emperror.With(errors.Errorf("create cluster error the returned status code is %d", resp.GetHttpStatus()), "cluster", a.context.Name)
	}

	// parse response
	var r ack.AlibabaClusterCreateResponse
	err = json.Unmarshal(resp.GetHttpContentBytes(), &r)
	if err != nil {
		return nil, emperror.With(err, "cluster", a.context.Name)
	}

	// We need this field to be able to implement the UndoAction for ClusterCreate
	a.context.ClusterID = r.ClusterID

	a.log.Infof("Alibaba cluster creating with id %s", r.ClusterID)

	// wait for cluster created
	a.log.Info("Waiting for cluster to get ready...")
	cluster, err := waitUntilClusterCreateOrScaleComplete(a.log, r.ClusterID, csClient, true)
	if err != nil {
		return nil, emperror.WrapWith(err, "cluster create failed", "cluster", a.context.Name)
	}

	return cluster, nil
}

// UndoAction rolls back this CreateACKClusterAction
func (a *CreateACKClusterAction) UndoAction() error {
	a.log.Info("EXECUTE UNDO CreateACKClusterAction")

	return emperror.With(deleteCluster(a.log, a.context.ClusterID, a.context.CSClient), "cluster", a.context.Name)
}
