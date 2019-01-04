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
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ess"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/pkg/cluster/acsk"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// UpdateACSKClusterAction describes the fields used across ACK cluster update operation
type UpdateACSKClusterAction struct {
	log       logrus.FieldLogger
	nodePools []*model.ACSKNodePoolModel
	context   *ACKContext
	region    string
}

// NewUpdateACSKClusterAction creates a new UpdateACSKClusterAction
func NewUpdateACSKClusterAction(log logrus.FieldLogger, nodepools []*model.ACSKNodePoolModel, clusterContext *ACKContext, region string) *UpdateACSKClusterAction {
	return &UpdateACSKClusterAction{
		log:       log,
		nodePools: nodepools,
		context:   clusterContext,
		region:    region,
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
	essClient := a.context.ESSClient

	attachInstanceIds := make([]string, 0)
	clusterInstances := make(map[string]string, 0)
	deleteInstances := make(map[string]string, 0)

	describeClusterNodesReq := cs.CreateDescribeClusterNodesRequest()
	describeClusterNodesReq.ClusterId = a.context.ClusterID
	describeClusterNodesReq.RegionId = a.region
	describeClusterNodesReq.SetScheme(requests.HTTPS)
	describeClusterNodesReq.SetDomain(acsk.AlibabaApiDomain)

	describeClusterNodesResp, err := csClient.DescribeClusterNodes(describeClusterNodesReq)
	if err != nil {
		return nil, err
	}

	if !describeClusterNodesResp.IsSuccess() || describeClusterNodesResp.GetHttpStatus() < 200 || describeClusterNodesResp.GetHttpStatus() > 299 {
		return nil, errors.Wrapf(err, "Unexpected http status code: %d", describeClusterNodesResp.GetHttpStatus())
	}

	var nodes acsk.AlibabaDescribeClusterNodesResponse

	err = json.Unmarshal(describeClusterNodesResp.GetHttpContentBytes(), &nodes)
	if err != nil {
		return nil, err
	}

	for _, node := range nodes.Nodes {
		clusterInstances[node.InstanceId] = node.IpAddress[0]
	}

	describeScalingConfReq := ess.CreateDescribeScalingConfigurationsRequest()
	describeScalingConfReq.RegionId = a.region
	describeScalingConfReq.SetDomain("ess." + a.region + ".aliyuncs.com")
	describeScalingConfReq.SetScheme(requests.HTTPS)

	describeScalingConfResp, err := essClient.DescribeScalingConfigurations(describeScalingConfReq)
	if err != nil {
		return nil, err
	}
	for _, conf := range describeScalingConfResp.ScalingConfigurations.ScalingConfiguration {
		for _, nodePool := range a.nodePools {
			if conf.ScalingConfigurationId == nodePool.ScalingConfId {

				pastInstanceIds := make([]string, 0)

				describeScalingInstancesRequest := ess.CreateDescribeScalingInstancesRequest()
				describeScalingInstancesRequest.SetScheme(requests.HTTPS)
				describeScalingInstancesRequest.SetDomain("ess." + a.region + ".aliyuncs.com")
				describeScalingInstancesRequest.SetContentType(requests.Json)

				describeScalingInstancesRequest.ScalingGroupId = nodePool.AsgId
				describeScalingInstancesRequest.ScalingConfigurationId = nodePool.ScalingConfId

				describeScalingInstancesResponse, err := essClient.DescribeScalingInstances(describeScalingInstancesRequest)
				if err != nil {
					return nil, err
				}

				for _, instance := range describeScalingInstancesResponse.ScalingInstances.ScalingInstance {
					pastInstanceIds = append(pastInstanceIds, instance.InstanceId)
				}

				modifyScalingGroupReq := ess.CreateModifyScalingGroupRequest()
				modifyScalingGroupReq.SetDomain("ess." + a.region + ".aliyuncs.com")
				modifyScalingGroupReq.SetScheme(requests.HTTPS)
				modifyScalingGroupReq.RegionId = a.region
				modifyScalingGroupReq.ScalingGroupId = nodePool.AsgId
				modifyScalingGroupReq.MinSize = requests.NewInteger(nodePool.MinCount)
				modifyScalingGroupReq.MaxSize = requests.NewInteger(nodePool.MaxCount)

				_, err = essClient.ModifyScalingGroup(modifyScalingGroupReq)
				if err != nil {
					return nil, err
				}

				availableInstanceIds, err := waitUntilScalingInstanceCreated(a.log, essClient, a.region, nodePool)
				if err != nil {
					return nil, err
				}

				for _, pastInstanceId := range pastInstanceIds {
					if !contains(availableInstanceIds, pastInstanceId) {
						for clusterInstanceId, clusterInstanceIp := range clusterInstances {
							if clusterInstanceId == pastInstanceId {
								deleteInstances[pastInstanceId] = clusterInstanceIp
							}
						}
					}
				}

				for _, availableInstanceId := range availableInstanceIds {
					if clusterInstances[availableInstanceId] == "" {
						attachInstanceIds = append(attachInstanceIds, availableInstanceId)
					}
				}
			}
		}
	}

	if len(attachInstanceIds) != 0 {
		a.log.Info("Attaching instances to cluster")
		attachInstanceRequest := cs.CreateAttachInstancesRequest()
		attachInstanceRequest.SetScheme(requests.HTTPS)
		attachInstanceRequest.SetDomain(acsk.AlibabaApiDomain)
		attachInstanceRequest.SetContentType(requests.Json)

		attachInstanceRequest.ClusterId = a.context.ClusterID

		content := map[string]interface{}{
			"instances": attachInstanceIds,
			"password":  "Hello1234", // Dummy password should be used here otherwise the api will fail
		}
		contentJSON, err := json.Marshal(content)
		if err != nil {
			return nil, err
		}
		attachInstanceRequest.SetContent(contentJSON)

		_, err = csClient.AttachInstances(attachInstanceRequest)
		if err != nil {
			return nil, err
		}

		if len(deleteInstances) == 0 {
			a.log.Info("Wait for instance attach")

			cluster, err := waitUntilClusterCreateOrScaleComplete(a.log, a.context.ClusterID, csClient, false)
			if err != nil {
				return nil, errors.Wrap(err, "cluster scale failed")
			}

			return cluster, nil
		}
	}

	if len(deleteInstances) != 0 {
		a.log.Info("remove instances from cluster")
		deleteClusterNodeReq := cs.CreateDeleteClusterNodeRequest()
		deleteClusterNodeReq.SetScheme(requests.HTTPS)
		deleteClusterNodeReq.SetDomain(acsk.AlibabaApiDomain)
		deleteClusterNodeReq.RegionId = a.region
		deleteClusterNodeReq.ClusterId = a.context.ClusterID
		for id, ip := range deleteInstances {
			deleteClusterNodeReq.Ip = ip

			content := map[string]interface{}{
				"node_id": id,
			}

			contentJSON, err := json.Marshal(content)
			if err != nil {
				return nil, err
			}
			deleteClusterNodeReq.SetContent(contentJSON)

			_, err = csClient.DeleteClusterNode(deleteClusterNodeReq)
			if err != nil {
				return nil, err
			}
		}

		a.log.Info("wait for cluster scale")

		cluster, err := waitUntilClusterCreateOrScaleComplete(a.log, a.context.ClusterID, csClient, false)
		if err != nil {
			return nil, errors.Wrap(err, "cluster scale failed")
		}

		return cluster, nil
	}

	r, err := getClusterDetails(a.context.ClusterID, csClient)
	if err != nil {
		return nil, err
	}

	return r, nil
}
