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
	context *ACSKClusterCreateContext
	log     logrus.FieldLogger
}

// NewCreateACSKNodePoolAction creates a new CreateACSKNodePoolAction
func NewCreateACSKNodePoolAction(log logrus.FieldLogger, creationContext *ACSKClusterCreateContext) *CreateACSKNodePoolAction {
	return &CreateACSKNodePoolAction{
		context: creationContext,
		log:     log,
	}
}

// GetName returns the name of this CreateACSKNodePoolAction
func (a *CreateACSKNodePoolAction) GetName() string {
	return "CreateACSKNodePoolAction"
}

// ExecuteAction executes this CreateACSKNodePoolAction
func (a *CreateACSKNodePoolAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	a.log.Infoln("EXECUTE CreateACSKNodePoolAction, cluster name", a.context.Name)

	cluster, ok := input.(*acsk.AlibabaDescribeClusterResponse)
	if !ok {
		err = errors.New("invalid input")

		return
	}

	errChan := make(chan error, len(a.context.NodePools))
	instanceIdsChan := make(chan []string, len(a.context.NodePools))
	defer close(errChan)
	defer close(instanceIdsChan)

	for _, nodePool := range a.context.NodePools {
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

			scalingGroupID := createScalingGroupResponse.ScalingGroupId

			nodePool.AsgId = scalingGroupID
			a.log.Infof("Scaling Group with id %s successfully created", scalingGroupID)
			a.log.Infof("Creating scaling configuration for group %s", scalingGroupID)

			scalingConfigurationRequest := ess.CreateCreateScalingConfigurationRequest()
			scalingConfigurationRequest.SetScheme(requests.HTTPS)
			scalingConfigurationRequest.SetDomain("ess." + cluster.RegionID + ".aliyuncs.com")
			scalingConfigurationRequest.SetContentType(requests.Json)

			scalingConfigurationRequest.ScalingGroupId = scalingGroupID
			scalingConfigurationRequest.SecurityGroupId = cluster.SecurityGroupID
			scalingConfigurationRequest.KeyPairName = a.context.Name
			scalingConfigurationRequest.InstanceType = nodePool.InstanceType
			scalingConfigurationRequest.SystemDiskCategory = "cloud_efficiency"
			scalingConfigurationRequest.ImageId = "centos_7_04_64_20G_alibase_20180419.vhd"
			scalingConfigurationRequest.Tags =
				fmt.Sprintf(`{"pipeline-created":"true","pipeline-cluster":"%s","pipeline-nodepool":"%s"`,
					a.context.AlibabaClusterCreateParams.Name, nodePool.Name)

			createConfigurationResponse, err := a.context.ESSClient.CreateScalingConfiguration(scalingConfigurationRequest)
			if err != nil {
				errChan <- err
				instanceIdsChan <- nil
				return
			}

			scalingConfID := createConfigurationResponse.ScalingConfigurationId

			nodePool.ScalingConfId = scalingConfID

			a.log.Infof("Scaling Configuration successfully created for group %s", scalingGroupID)

			enableSGRequest := ess.CreateEnableScalingGroupRequest()
			enableSGRequest.SetScheme(requests.HTTPS)
			enableSGRequest.SetDomain("ess." + cluster.RegionID + ".aliyuncs.com")
			enableSGRequest.SetContentType(requests.Json)

			enableSGRequest.ScalingGroupId = scalingGroupID
			enableSGRequest.ActiveScalingConfigurationId = scalingConfID

			_, err = a.context.ESSClient.EnableScalingGroup(enableSGRequest)
			if err != nil {
				errChan <- err
				instanceIdsChan <- nil
				return
			}

			instanceIds, err := waitUntilScalingInstanceCreated(a.log, a.context.ESSClient, cluster.RegionID, scalingGroupID, scalingConfID)
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
	for i := 0; i < len(a.context.NodePools); i++ {
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
		return nil, emperror.WrapWith(err, "nodepool creation failed", "clusterName", a.context.Name)
	}

	return clusterWithPools, err
}

// UndoAction rolls back this CreateACSKNodePoolAction
func (a *CreateACSKNodePoolAction) UndoAction() (err error) {
	a.log.Info("EXECUTE UNDO CreateACSKNodePoolAction")

	errChan := make(chan error, len(a.context.NodePools))
	defer close(errChan)

	for _, nodePool := range a.context.NodePools {
		go func(nodePool *model.ACSKNodePoolModel) {

			deleteSGRequest := ess.CreateDeleteScalingGroupRequest()
			deleteSGRequest.SetScheme(requests.HTTPS)
			deleteSGRequest.SetDomain("ess." + a.context.AlibabaClusterCreateParams.RegionID + ".aliyuncs.com")
			deleteSGRequest.SetContentType(requests.Json)
			if nodePool.AsgId == "" {
				// Asg could not be created nothing to remove
				errChan <- nil
				return
			}

			deleteSGRequest.ScalingGroupId = nodePool.AsgId
			deleteSGRequest.ForceDelete = requests.NewBoolean(true)

			_, err := a.context.ESSClient.DeleteScalingGroup(deleteSGRequest)
			if err != nil {
				errChan <- err
				return
			}

			errChan <- nil
		}(nodePool)
	}

	for i := 0; i < len(a.context.NodePools); i++ {
		e := <-errChan
		if e != nil {
			a.log.Error(e)

			err = e
		}
	}

	return err
}
