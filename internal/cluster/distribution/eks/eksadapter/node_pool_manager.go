// Copyright © 2020 Banzai Cloud
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

package eksadapter

import (
	"context"
	"fmt"
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"go.uber.org/cadence/client"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksworkflow"
	"github.com/banzaicloud/pipeline/pkg/kubernetes/custom/npls"
	sdkCloudFormation "github.com/banzaicloud/pipeline/pkg/sdk/providers/amazon/cloudformation"
)

const (
	// nodePoolStackNamePrefix is the prefix of CloudFormation stack names of
	// node pools managed by the pipeline.
	nodePoolStackNamePrefix = "pipeline-eks-nodepool-"
)

type nodePoolManager struct {
	awsFactory            workflow.AWSFactory
	cloudFormationFactory workflow.CloudFormationAPIFactory
	dynamicClientFactory  cluster.DynamicKubeClientFactory
	enterprise            bool
	namespace             string
	workflowClient        client.Client
}

// NewNodePoolManager returns a new eks.NodePoolManager
// that manages node pools asynchronously via Cadence workflows.
func NewNodePoolManager(
	awsFactory workflow.AWSFactory,
	cloudFormationFactory workflow.CloudFormationAPIFactory,
	dynamicClientFactory cluster.DynamicKubeClientFactory,
	enterprise bool,
	namespace string,
	workflowClient client.Client,
) eks.NodePoolManager {
	return nodePoolManager{
		awsFactory:            awsFactory,
		cloudFormationFactory: cloudFormationFactory,
		dynamicClientFactory:  dynamicClientFactory,
		enterprise:            enterprise,
		namespace:             namespace,
		workflowClient:        workflowClient,
	}
}

func (n nodePoolManager) UpdateNodePool(
	ctx context.Context,
	c cluster.Cluster,
	nodePoolName string,
	nodePoolUpdate eks.NodePoolUpdate,
) (string, error) {
	taskList := "pipeline"
	if n.enterprise {
		taskList = "pipeline-enterprise"
	}

	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     taskList,
		ExecutionStartToCloseTimeout: 30 * 24 * 60 * time.Minute,
	}

	input := eksworkflow.UpdateNodePoolWorkflowInput{
		ProviderSecretID: c.SecretID.String(),
		Region:           c.Location,

		StackName: generateNodePoolStackName(c.Name, nodePoolName),

		ClusterID:       c.ID,
		ClusterSecretID: c.ConfigSecretID.String(),
		ClusterName:     c.Name,
		NodePoolName:    nodePoolName,
		OrganizationID:  c.OrganizationID,

		NodeVolumeSize: nodePoolUpdate.VolumeSize,
		NodeImage:      nodePoolUpdate.Image,

		Options: eks.NodePoolUpdateOptions{
			MaxSurge:       nodePoolUpdate.Options.MaxSurge,
			MaxBatchSize:   nodePoolUpdate.Options.MaxBatchSize,
			MaxUnavailable: nodePoolUpdate.Options.MaxUnavailable,
			Drain: eks.NodePoolUpdateDrainOptions{
				Timeout:     nodePoolUpdate.Options.Drain.Timeout,
				FailOnError: nodePoolUpdate.Options.Drain.FailOnError,
				PodSelector: nodePoolUpdate.Options.Drain.PodSelector,
			},
		},
		ClusterTags: c.Tags,
	}

	e, err := n.workflowClient.StartWorkflow(ctx, workflowOptions, eksworkflow.UpdateNodePoolWorkflowName, input)
	if err != nil {
		return "", errors.WrapWithDetails(err, "failed to start workflow", "workflow", eksworkflow.UpdateNodePoolWorkflowName)
	}

	return e.ID, nil
}

// TODO: this is temporary
func generateNodePoolStackName(clusterName string, poolName string) string {
	return nodePoolStackNamePrefix + clusterName + "-" + poolName
}

// ListNodePools lists node pools from a cluster.
func (n nodePoolManager) ListNodePools(ctx context.Context, cluster cluster.Cluster, nodePoolNames []string) ([]eks.NodePool, error) {
	clusterClient, err := n.dynamicClientFactory.FromSecret(ctx, cluster.ConfigSecretID.String())
	if err != nil {
		return nil, errors.WrapWithDetails(err, "creating dynamic Kubernetes client factory failed", "cluster", cluster)
	}

	manager := npls.NewManager(clusterClient, n.namespace)
	labelSets, err := manager.GetAll(ctx)
	if err != nil {
		return nil, errors.WrapWithDetails(err, "retrieving node pool label sets failed",
			"cluster", cluster,
			"namespace", n.namespace,
		)
	}

	awsClient, err := n.awsFactory.New(cluster.OrganizationID, cluster.SecretID.ResourceID, cluster.Location)
	if err != nil {
		return nil, errors.WrapWithDetails(err, "creating aws factory failed", "cluster", cluster)
	}

	cfClient := n.cloudFormationFactory.New(awsClient)
	describeStacksInput := cloudformation.DescribeStacksInput{}
	nodePools := make([]eks.NodePool, 0, len(nodePoolNames))
	for _, nodePoolName := range nodePoolNames {
		nodePools = append(nodePools, eks.NodePool{
			Name:   nodePoolName,
			Labels: labelSets[nodePoolName],
		})
		nodePool := &nodePools[len(nodePools)-1]

		stackName := generateNodePoolStackName(cluster.Name, nodePoolName)
		describeStacksInput.StackName = &stackName
		stackDescriptions, err := cfClient.DescribeStacks(&describeStacksInput)
		if err != nil {
			nodePool.Status = eks.NodePoolStatusUnknown
			nodePool.StatusMessage = fmt.Sprintf("Retrieving node pool information failed: %s", err)

			continue
		} else if len(stackDescriptions.Stacks) == 0 {
			nodePool.Status = eks.NodePoolStatusUnknown
			nodePool.StatusMessage = "Retrieving node pool information failed: node pool not found."

			continue
		}

		stack := stackDescriptions.Stacks[0]

		var nodePoolParameters struct {
			ClusterAutoscalerEnabled    bool   `mapstructure:"ClusterAutoscalerEnabled"`
			NodeAutoScalingGroupMaxSize int    `mapstructure:"NodeAutoScalingGroupMaxSize"`
			NodeAutoScalingGroupMinSize int    `mapstructure:"NodeAutoScalingGroupMinSize"`
			NodeAutoScalingInitSize     int    `mapstructure:"NodeAutoScalingInitSize"`
			NodeImageID                 string `mapstructure:"NodeImageId"`
			NodeInstanceType            string `mapstructure:"NodeInstanceType"`
			NodeSpotPrice               string `mapstructure:"NodeSpotPrice"`
			NodeVolumeSize              int    `mapstructure:"NodeVolumeSize"`
			Subnets                     string `mapstructure:"Subnets"`
		}

		err = sdkCloudFormation.ParseStackParameters(stack.Parameters, &nodePoolParameters)
		if err != nil {
			nodePool.Status = eks.NodePoolStatusError
			nodePool.StatusMessage = "Retrieving node pool information failed: invalid CloudFormation stack parameters."

			continue
		}

		nodePool.Size = nodePoolParameters.NodeAutoScalingInitSize
		nodePool.Autoscaling = eks.Autoscaling{
			Enabled: nodePoolParameters.ClusterAutoscalerEnabled,
			MinSize: nodePoolParameters.NodeAutoScalingGroupMinSize,
			MaxSize: nodePoolParameters.NodeAutoScalingGroupMaxSize,
		}
		nodePool.VolumeSize = nodePoolParameters.NodeVolumeSize
		nodePool.InstanceType = nodePoolParameters.NodeInstanceType
		nodePool.Image = nodePoolParameters.NodeImageID
		nodePool.SpotPrice = nodePoolParameters.NodeSpotPrice
		nodePool.SubnetID = nodePoolParameters.Subnets // Note: currently we ensure a single value at creation.
		nodePool.Status = eks.NewNodePoolStatusFromCFStackStatus(aws.StringValue(stack.StackStatus))
		nodePool.StatusMessage = aws.StringValue(stack.StackStatusReason)
	}

	return nodePools, nil
}
