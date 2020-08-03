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
	"strconv"
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

		NodeImage: nodePoolUpdate.Image,

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
		stackName := generateNodePoolStackName(cluster.Name, nodePoolName)
		describeStacksInput.StackName = &stackName
		stackDescriptions, err := cfClient.DescribeStacks(&describeStacksInput)
		if err != nil {
			return nil, errors.WrapWithDetails(err, "retrieving node pool cloudformation stack failed",
				"cluster", cluster,
				"input", describeStacksInput,
			)
		} else if len(stackDescriptions.Stacks) == 0 {
			return nil, errors.NewWithDetails("missing required node pool cloudformation stack",
				"cluster", cluster,
				"stackName", stackName,
			)
		}

		stack := stackDescriptions.Stacks[0]

		parameterMap := make(map[string]string, len(stack.Parameters))
		for _, parameter := range stack.Parameters {
			parameterMap[aws.StringValue(parameter.ParameterKey)] = aws.StringValue(parameter.ParameterValue)
		}

		var clusterAutoscalerEnabled bool
		var nodeAutoScalingGroupMaxSize int
		var nodeAutoScalingGroupMinSize int
		var nodeAutoScalingInitSize int
		nodePoolParameters := map[string]interface{}{
			"ClusterAutoscalerEnabled":    &clusterAutoscalerEnabled,
			"NodeAutoScalingGroupMaxSize": &nodeAutoScalingGroupMaxSize,
			"NodeAutoScalingGroupMinSize": &nodeAutoScalingGroupMinSize,
			"NodeAutoScalingInitSize":     &nodeAutoScalingInitSize,
		}

		err = parseStackParameters(parameterMap, nodePoolParameters)
		if err != nil {
			return nil, errors.WrapWithDetails(err, "parsing node pool stack parameters failed",
				"stackName", stackName)
		}

		nodePool := eks.NodePool{
			Name:   nodePoolName,
			Labels: labelSets[nodePoolName],
			Size:   nodeAutoScalingInitSize,
			Autoscaling: eks.Autoscaling{
				Enabled: clusterAutoscalerEnabled,
				MinSize: nodeAutoScalingGroupMinSize,
				MaxSize: nodeAutoScalingGroupMaxSize,
			},
			InstanceType: parameterMap["NodeInstanceType"],
			Image:        parameterMap["NodeImageId"],
			SpotPrice:    parameterMap["NodeSpotPrice"],
		}

		nodePools = append(nodePools, nodePool)
	}

	return nodePools, nil
}

func parseStackParameters(parameterMap map[string]string, resultPointerMap map[string]interface{}) (err error) {
	parseErrors := make([]error, 0)
	for parameterKey, resultPointer := range resultPointerMap {
		parameterRawValue, isExisting := parameterMap[parameterKey]
		if !isExisting {
			parseErrors = append(parseErrors, errors.NewWithDetails("missing stack parameter",
				"parameterKey", parameterKey))
		}

		err = parseStringValue(parameterRawValue, resultPointer)
		if err != nil {
			parseErrors = append(parseErrors, errors.WrapWithDetails(err, "parsing node pool cloudformation stack parameter failed",
				"parameterKey", parameterKey))
		}
	}

	if len(parseErrors) != 0 {
		return errors.Combine(parseErrors...)
	}

	return nil
}

// parseStringValue parses a string value to a strongly typed target result
// object or returns error on failure.
func parseStringValue(rawValue string, resultPointer interface{}) (err error) {
	switch typedPointer := resultPointer.(type) {
	case *bool:
		if typedPointer == nil {
			return errors.NewWithDetails("parsing raw string value received nil result pointer", "type", "bool")
		}

		*typedPointer, err = strconv.ParseBool(rawValue)
		if err != nil {
			return errors.NewWithDetails("parsing raw string value failed", "rawValue", rawValue, "type", "bool")
		}
	case *int:
		if typedPointer == nil {
			return errors.NewWithDetails("parsing raw string value received nil result pointer", "type", "int")
		}

		*typedPointer, err = strconv.Atoi(rawValue)
		if err != nil {
			return errors.NewWithDetails("parsing raw string value failed", "rawValue", rawValue, "type", "int")
		}
	default:
		return errors.NewWithDetails("parsing raw string value type not implemented")
	}

	return nil
}
