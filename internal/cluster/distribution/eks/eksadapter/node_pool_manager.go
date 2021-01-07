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
	"sort"
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"go.uber.org/cadence/client"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksworkflow"
	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
	"github.com/banzaicloud/pipeline/pkg/kubernetes/custom/npls"
)

const (
	// nodePoolStackNamePrefix is the prefix of CloudFormation stack names of
	// node pools managed by the pipeline.
	nodePoolStackNamePrefix = "pipeline-eks-nodepool-"
)

type nodePoolManager struct {
	awsFactory            awsworkflow.AWSFactory
	cloudFormationFactory awsworkflow.CloudFormationAPIFactory
	dynamicClientFactory  cluster.DynamicKubeClientFactory
	enterprise            bool
	getUserID             func(ctx context.Context) (userID uint)
	namespace             string
	workflowClient        client.Client
}

// NewNodePoolManager returns a new eks.NodePoolManager
// that manages node pools asynchronously via Cadence workflows.
func NewNodePoolManager(
	awsFactory awsworkflow.AWSFactory,
	cloudFormationFactory awsworkflow.CloudFormationAPIFactory,
	dynamicClientFactory cluster.DynamicKubeClientFactory,
	enterprise bool,
	getUserID func(ctx context.Context) (userID uint),
	namespace string,
	workflowClient client.Client,
) eks.NodePoolManager {
	return nodePoolManager{
		awsFactory:            awsFactory,
		cloudFormationFactory: cloudFormationFactory,
		dynamicClientFactory:  dynamicClientFactory,
		enterprise:            enterprise,
		getUserID:             getUserID,
		namespace:             namespace,
		workflowClient:        workflowClient,
	}
}

// CreateNodePool initiates the node pool creation process.
//
// Implements the eks.NodePoolManager interface.
func (n nodePoolManager) CreateNodePool(ctx context.Context, c cluster.Cluster, nodePool eks.NewNodePool) (err error) {
	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: 30 * 24 * 60 * time.Minute,
	}

	input := workflow.CreateNodePoolWorkflowInput{
		ClusterID:                    c.ID,
		NodePool:                     nodePool,
		NodePoolSubnetIDs:            []string{nodePool.SubnetID},
		ShouldCreateNodePoolLabelSet: true,
		ShouldStoreNodePool:          true,
		ShouldUpdateClusterStatus:    true,
		CreatorUserID:                n.getUserID(ctx),
	}

	_, err = n.workflowClient.StartWorkflow(ctx, workflowOptions, workflow.CreateNodePoolWorkflowName, input)
	if err != nil {
		return errors.WrapWithDetails(err, "failed to start workflow", "workflow", workflow.CreateNodePoolWorkflowName)
	}

	return nil
}

// DeleteNodePool deletes an existing node pool in a cluster.
func (n nodePoolManager) DeleteNodePool(
	ctx context.Context, c cluster.Cluster, existingNodePool eks.ExistingNodePool, shouldUpdateClusterStatus bool,
) (err error) {
	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: 30 * 24 * 60 * time.Minute,
	}

	input := workflow.DeleteNodePoolWorkflowInput{
		ClusterID:                 c.ID,
		ClusterName:               c.Name,
		NodePoolName:              existingNodePool.Name,
		OrganizationID:            c.OrganizationID,
		Region:                    c.Location,
		SecretID:                  c.SecretID.ResourceID,
		ShouldUpdateClusterStatus: shouldUpdateClusterStatus,
	}

	_, err = n.workflowClient.StartWorkflow(ctx, workflowOptions, workflow.DeleteNodePoolWorkflowName, input)
	if err != nil {
		return errors.WrapWithDetails(err, "failed to start workflow", "workflow", workflow.DeleteNodePoolWorkflowName)
	}

	return nil
}

// getLabelSets retrieves the Kubernetes label sets of the node pools.
func (n nodePoolManager) getLabelSets(
	ctx context.Context, c cluster.Cluster,
) (labelSets map[string]map[string]string, err error) {
	clusterClient, err := n.dynamicClientFactory.FromSecret(ctx, c.ConfigSecretID.String())
	if err != nil {
		return nil, errors.WrapWithDetails(err, "creating dynamic Kubernetes client factory failed",
			"configSecretId", c.ConfigSecretID.String(),
		)
	}

	manager := npls.NewManager(clusterClient, n.namespace)
	labelSets, err = manager.GetAll(ctx)
	if err != nil {
		return nil, errors.WrapWithDetails(err, "listing node pool label sets failed",
			"namespace", n.namespace,
		)
	}

	return labelSets, nil
}

// ListNodePools lists node pools from a cluster.
func (n nodePoolManager) ListNodePools(
	ctx context.Context,
	c cluster.Cluster,
	existingNodePools map[string]eks.ExistingNodePool,
) (nodePools []eks.NodePool, err error) {
	if c.ConfigSecretID.ResourceID == "" || // Note: cluster is being created or errorred before k8s secret would be available.
		c.Status == cluster.Deleting {
		return nil, cluster.NotReadyError{
			OrganizationID: c.OrganizationID,
			ID:             c.ID,
			Name:           c.Name,
		}
	}

	labelSets, err := n.getLabelSets(ctx, c)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "retrieving node pool label sets failed",
			"organizationId", c.OrganizationID,
			"clusterId", c.ID,
			"clusterName", c.Name,
		)
	}

	cloudFormationClient, err := n.newCloudFormationClient(ctx, c)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "instantiating CloudFormation client failed",
			"organizationId", c.OrganizationID,
			"clusterId", c.ID,
			"clusterName", c.Name,
		)
	}

	nodePools = make([]eks.NodePool, 0, len(nodePools))
	for _, existingNodePool := range existingNodePools {
		nodePools = append(nodePools, newNodePoolFromCloudFormation(
			cloudFormationClient,
			existingNodePool,
			generateNodePoolStackName(c.Name, existingNodePool.Name),
			labelSets[existingNodePool.Name],
		))
	}

	sort.Slice(nodePools, func(firstIndex, secondIndex int) (isLessThan bool) {
		return nodePools[firstIndex].Name < nodePools[secondIndex].Name
	})

	return nodePools, nil
}

// newCloudFormationClient instantiates a CloudFormation client from the
// manager's factories.
func (n nodePoolManager) newCloudFormationClient(
	ctx context.Context, c cluster.Cluster,
) (cloudFormationClient cloudformationiface.CloudFormationAPI, err error) {
	awsClient, err := n.awsFactory.New(c.OrganizationID, c.SecretID.ResourceID, c.Location)
	if err != nil {
		return nil, errors.WrapWithDetails(err, "creating aws factory failed",
			"organizationId", c.OrganizationID,
			"location", c.Location,
			"secretId", c.SecretID.ResourceID,
		)
	}

	return n.cloudFormationFactory.New(awsClient), nil
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

		NodeVolumeEncryption: nodePoolUpdate.VolumeEncryption,
		NodeVolumeSize:       nodePoolUpdate.VolumeSize,
		NodeImage:            nodePoolUpdate.Image,
		SecurityGroups:       nodePoolUpdate.SecurityGroups,
		UseInstanceStore:     nodePoolUpdate.UseInstanceStore,

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

// newNodePoolFromCloudFormation tries to describe the node pool's stack and
// return all available information about it or a descriptive status and status
// message.
func newNodePoolFromCloudFormation(
	cfClient cloudformationiface.CloudFormationAPI,
	existingNodePool eks.ExistingNodePool,
	stackName string, // Note: temporary until we eliminate stack name usage.
	labels map[string]string,
) (nodePool eks.NodePool) {
	stackIdentifier := existingNodePool.StackID
	if stackIdentifier == "" { // Note: CloudFormation stack creation not started yet.
		stackIdentifier = stackName
	}

	stackDescriptions, err := cfClient.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(stackIdentifier),
	})
	if err != nil {
		return eks.NewNodePoolFromCFStackDescriptionError(err, existingNodePool)
	} else if len(stackDescriptions.Stacks) == 0 {
		return eks.NewNodePoolWithNoValues(
			existingNodePool.Name,
			eks.NodePoolStatusUnknown,
			"retrieving node pool information failed: node pool not found",
		)
	} else if len(stackDescriptions.Stacks) > 1 {
		return eks.NewNodePoolWithNoValues(
			existingNodePool.Name,
			eks.NodePoolStatusUnknown,
			"retrieving node pool information failed: multiple node pools found",
		)
	}

	return eks.NewNodePoolFromCFStack(existingNodePool.Name, labels, stackDescriptions.Stacks[0])
}
