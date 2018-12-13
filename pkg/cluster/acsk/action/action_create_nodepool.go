package action

import (
	"encoding/json"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ess"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/pkg/cluster/acsk"
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
	defer close(errChan)

	for _, nodePool := range a.context.NodePools {
		go func(nodePool *model.ACSKNodePoolModel) {
			request := ess.CreateCreateScalingGroupRequest()
			request.SetScheme(requests.HTTPS)

			a.log.WithFields(logrus.Fields{
				"region":        cluster.RegionID,
				"zone":          cluster.ZoneID,
				"instance_type": nodePool.InstanceType,
			}).Info("creating scaling group")

			request.RegionId = cluster.RegionID
			request.MinSize = requests.NewInteger(nodePool.MinCount)
			request.MaxSize = requests.NewInteger(nodePool.MaxCount)

			response, err := a.context.ESSClient.CreateScalingGroup(request)
			if err != nil {
				errChan <- err
				return
			}




			var instanceIds []string
			for i := 0; i < nodePool.Count; i++ {
				request := ecs.CreateRunInstancesRequest()
				request.SetScheme(requests.HTTPS)

				a.log.WithFields(logrus.Fields{
					"region":        cluster.RegionID,
					"zone":          cluster.ZoneID,
					"instance_type": nodePool.InstanceType,
					"system_disk":   nodePool.SystemDiskCategory,
				}).Info("creating instance")

				request.RegionId = cluster.RegionID
				request.ZoneId = cluster.ZoneID
				request.VSwitchId = cluster.VSwitchID
				request.SecurityGroupId = cluster.SecurityGroupID
				request.KeyPairName = a.context.AlibabaClusterCreateParams.KeyPair
				//request.ImageId = nodePool.Image
				// TODO: choose an image from the region
				request.ImageId = "centos_7_04_64_20G_alibase_20180419.vhd"
				request.InstanceType = nodePool.InstanceType
				request.SystemDiskCategory = nodePool.SystemDiskCategory
				request.SystemDiskSize = "30"
				request.IoOptimized = "optimized"
				//request.Password = "Hello1234"
				request.Tag = &[]ecs.RunInstancesTag{
					{
						Key:   "pipeline-created",
						Value: "true",
					},
					{
						Key:   "pipeline-cluster",
						Value: a.context.AlibabaClusterCreateParams.Name,
					},
					{
						Key:   "pipeline-nodepool",
						Value: nodePool.Name,
					},
				}

				response, err := a.context.ECSClient.RunInstances(request)
				if err != nil {
					errChan <- err
					return
				}

				instanceIds =response.InstanceIdSets.InstanceIdSet
			}

			// TODO: implement proper node checking
			// this is an optimistic estimate
			time.Sleep(30 * time.Second)

			request := cs.CreateAttachInstancesRequest()
			request.ClusterId = cluster.ClusterID
			request.SetScheme(requests.HTTPS)
			request.SetDomain("cs.aliyuncs.com")
			request.SetContentType("application/json")
			content := map[string]interface{}{
				"instances": instanceIds,
				"password":  "Hello1234",
			}
			contentJSON, err := json.Marshal(content)
			if err != nil {
				errChan <- err
				return
			}
			request.SetContent(contentJSON)

			resp, err := a.context.CSClient.AttachInstances(request)
			a.log.Info(resp)
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

	return
}

// UndoAction rolls back this CreateACSKNodePoolAction
func (a *CreateACSKNodePoolAction) UndoAction() (err error) {
	a.log.Info("EXECUTE UNDO CreateACSKNodePoolAction")

	errChan := make(chan error, len(a.context.NodePools))
	defer close(errChan)

	for _, nodePool := range a.context.NodePools {
		go func(nodePool *model.ACSKNodePoolModel) {
			for i := 0; i < nodePool.Count; i++ {
				listRequest := ecs.CreateDescribeInstancesRequest()
				listRequest.Tag = &[]ecs.DescribeInstancesTag{
					{
						Key:   "pipeline-created",
						Value: "true",
					},
					{
						Key:   "pipeline-cluster",
						Value: a.context.AlibabaClusterCreateParams.Name,
					},
					{
						Key:   "pipeline-nodepool",
						Value: nodePool.Name,
					},
				}

				listResponse, err := a.context.ECSClient.DescribeInstances(listRequest)
				if err != nil {
					errChan <- err
					return
				}

				// TODO: handle pagination
				for _, instance := range listResponse.Instances.Instance {
					stopRequest := ecs.CreateStopInstanceRequest()
					stopRequest.InstanceId = instance.InstanceId

					_, err := a.context.ECSClient.StopInstance(stopRequest)
					if err != nil {
						errChan <- err
						return
					}

					// TODO: check if the instance is stopped
					// this timeout is an optimistic estimation
					time.Sleep(10 * time.Second)

					deleteRequest := ecs.CreateDeleteInstanceRequest()
					deleteRequest.InstanceId = instance.InstanceId

					_, err = a.context.ECSClient.DeleteInstance(deleteRequest)
					if err != nil {
						errChan <- err
						return
					}
				}
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
