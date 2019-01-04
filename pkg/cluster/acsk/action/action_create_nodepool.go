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
	"fmt"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ess"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/pkg/cluster/acsk"
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
func (a *CreateACSKNodePoolAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	cluster, ok := input.(*acsk.AlibabaDescribeClusterResponse)
	if !ok {
		err = errors.New("invalid input")

		return
	}
	a.log.Infoln("EXECUTE CreateACSKNodePoolAction, cluster name", cluster.Name)

	if len(a.nodePools) == 0 {
		a.log.Info("no new nodepools in the request")
		r, err := getClusterDetails(a.context.ClusterID, a.context.CSClient)
		if err != nil {
			return nil, err
		}

		return r, nil
	}

	errChan := make(chan error, len(a.nodePools))
	instanceIdsChan := make(chan []string, len(a.nodePools))
	defer close(errChan)
	defer close(instanceIdsChan)

	for _, nodePool := range a.nodePools {
		go func(nodePool *model.ACSKNodePoolModel) {
			scalingGroupRequest := ess.CreateCreateScalingGroupRequest()
			scalingGroupRequest.SetScheme(requests.HTTPS)
			scalingGroupRequest.SetDomain("ess." + cluster.RegionID + ".aliyuncs.com")
			scalingGroupRequest.SetContentType(requests.Json)

			a.log.WithFields(logrus.Fields{
				"region":        cluster.RegionID,
				"zone":          cluster.ZoneID,
				"instance_type": nodePool.InstanceType,
			}).Info("creating scaling group")

			scalingGroupRequest.MinSize = requests.NewInteger(nodePool.MinCount)
			scalingGroupRequest.MaxSize = requests.NewInteger(nodePool.MaxCount)
			scalingGroupRequest.VSwitchId = cluster.VSwitchID
			scalingGroupRequest.ScalingGroupName = fmt.Sprintf("asg-%s-%s", nodePool.Name, cluster.ClusterID)

			createScalingGroupResponse, err := a.context.ESSClient.CreateScalingGroup(scalingGroupRequest)
			if err != nil {
				errChan <- err
				instanceIdsChan <- nil
				return
			}

			nodePool.AsgId = createScalingGroupResponse.ScalingGroupId
			a.log.Infof("Scaling Group with id %s successfully created", nodePool.AsgId)
			a.log.Infof("Creating scaling configuration for group %s", nodePool.AsgId)

			scalingConfigurationRequest := ess.CreateCreateScalingConfigurationRequest()
			scalingConfigurationRequest.SetScheme(requests.HTTPS)
			scalingConfigurationRequest.SetDomain("ess." + cluster.RegionID + ".aliyuncs.com")
			scalingConfigurationRequest.SetContentType(requests.Json)

			scalingConfigurationRequest.ScalingGroupId = nodePool.AsgId
			scalingConfigurationRequest.SecurityGroupId = cluster.SecurityGroupID
			scalingConfigurationRequest.KeyPairName = cluster.Name
			scalingConfigurationRequest.InstanceType = nodePool.InstanceType
			scalingConfigurationRequest.SystemDiskCategory = "cloud_efficiency"
			scalingConfigurationRequest.ImageId = "centos_7_04_64_20G_alibase_20180419.vhd"
			scalingConfigurationRequest.Tags =
				fmt.Sprintf(`{"pipeline-created":"true","pipeline-cluster":"%s","pipeline-nodepool":"%s"`,
					cluster.Name, nodePool.Name)

			createConfigurationResponse, err := a.context.ESSClient.CreateScalingConfiguration(scalingConfigurationRequest)
			if err != nil {
				errChan <- err
				instanceIdsChan <- nil
				return
			}

			nodePool.ScalingConfId = createConfigurationResponse.ScalingConfigurationId

			a.log.Infof("Scaling Configuration successfully created for group %s", nodePool.AsgId)

			enableSGRequest := ess.CreateEnableScalingGroupRequest()
			enableSGRequest.SetScheme(requests.HTTPS)
			enableSGRequest.SetDomain("ess." + cluster.RegionID + ".aliyuncs.com")
			enableSGRequest.SetContentType(requests.Json)

			enableSGRequest.ScalingGroupId = nodePool.AsgId
			enableSGRequest.ActiveScalingConfigurationId = nodePool.ScalingConfId

			_, err = a.context.ESSClient.EnableScalingGroup(enableSGRequest)
			if err != nil {
				errChan <- err
				instanceIdsChan <- nil
				return
			}

			instanceIds, err := waitUntilScalingInstanceCreated(a.log, a.context.ESSClient, cluster.RegionID, nodePool)
			if err != nil {
				errChan <- err
				instanceIdsChan <- nil
				return
			}

			errChan <- nil
			instanceIdsChan <- instanceIds
		}(nodePool)
	}

	var instanceIds []string
	for i := 0; i < len(a.nodePools); i++ {
		e := <-errChan
		ids := <-instanceIdsChan
		if e != nil {
			a.log.Error(e)
			err = e
		} else {
			instanceIds = append(instanceIds, ids...)
		}
	}
	if err != nil {
		return
	}

	a.log.Info("Attaching nodepools to cluster")
	attachInstanceRequest := cs.CreateAttachInstancesRequest()
	attachInstanceRequest.SetScheme(requests.HTTPS)
	attachInstanceRequest.SetDomain(acsk.AlibabaApiDomain)
	attachInstanceRequest.SetContentType(requests.Json)

	attachInstanceRequest.ClusterId = cluster.ClusterID

	content := map[string]interface{}{
		"instances": instanceIds,
		"password":  "Hello1234", // Dummy password should be used here otherwise the api will fail
	}
	contentJSON, err := json.Marshal(content)
	if err != nil {
		return
	}
	attachInstanceRequest.SetContent(contentJSON)

	_, err = a.context.CSClient.AttachInstances(attachInstanceRequest)
	if err != nil {
		return
	}
	a.log.Info("Wait for nodepool attach")
	clusterWithPools, err := waitUntilClusterCreateOrScaleComplete(a.log, cluster.ClusterID, a.context.CSClient, false)
	if err != nil {
		return nil, emperror.WrapWith(err, "nodepool creation failed", "clusterName", cluster.Name)
	}

	return clusterWithPools, err
}

// UndoAction rolls back this CreateACSKNodePoolAction
func (a *CreateACSKNodePoolAction) UndoAction() (err error) {
	a.log.Info("EXECUTE UNDO CreateACSKNodePoolAction")
	return deleteNodepools(a.log, a.nodePools, a.context.ESSClient, a.region)
}
