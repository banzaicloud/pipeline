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

package pkeawsadapter

import (
	"context"
	"sort"
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence/client"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke/pkeaws"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke/pkeaws/pkeawsprovider/workflow"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke/pkeaws/pkeawsworkflow"
	"github.com/banzaicloud/pipeline/pkg/kubernetes/custom/npls"
)

type nodePoolManager struct {
	dynamicClientFactory cluster.DynamicKubeClientFactory
	enterprise           bool
	namespace            string
	workflowClient       client.Client
}

// NewNodePoolManager returns a new pke.NodePoolManager
// that manages node pools asynchronously via Cadence workflows.
func NewNodePoolManager(
	dynamicClientFactory cluster.DynamicKubeClientFactory,
	enterprise bool,
	namespace string,
	workflowClient client.Client,
) pke.NodePoolManager {
	return nodePoolManager{
		dynamicClientFactory: dynamicClientFactory,
		enterprise:           enterprise,
		namespace:            namespace,
		workflowClient:       workflowClient,
	}
}

// DeleteNodePool deletes an existing node pool in a cluster.
func (n nodePoolManager) DeleteNodePool(
	ctx context.Context, c cluster.Cluster, existingNodePool pke.ExistingNodePool, shouldUpdateClusterStatus bool,
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

// ListNodePools lists node pools from a cluster.
func (n nodePoolManager) ListNodePools(
	ctx context.Context,
	c cluster.Cluster,
	existingNodePools map[string]pke.ExistingNodePool,
) (nodePools []pke.NodePool, err error) {
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

	nodePools = make([]pke.NodePool, 0, len(nodePools))
	for _, existingNodePool := range existingNodePools {
		nodePools = append(nodePools, newNodePoolFromCloudFormation(
			cloudFormationClient,
			existingNodePool,
			pkeaws.GenerateNodePoolStackName(c.Name, existingNodePool.Name),
			labelSets[existingNodePool.Name],
		))
	}

	sort.Slice(nodePools, func(firstIndex, secondIndex int) (isLessThan bool) {
		return nodePools[firstIndex].Name < nodePools[secondIndex].Name
	})

	return nodePools, nil
}

func (n nodePoolManager) UpdateNodePool(
	ctx context.Context,
	c cluster.Cluster,
	nodePoolName string,
	nodePoolUpdate pke.NodePoolUpdate,
) (string, error) {
	taskList := "pipeline"
	if n.enterprise {
		taskList = "pipeline-enterprise"
	}

	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     taskList,
		ExecutionStartToCloseTimeout: 30 * 24 * 60 * time.Minute,
	}

	input := pkeawsworkflow.UpdateNodePoolWorkflowInput{
		ProviderSecretID: c.SecretID.String(),
		Region:           c.Location,

		StackName: pkeaws.GenerateNodePoolStackName(c.Name, nodePoolName),

		ClusterID:       c.ID,
		ClusterSecretID: c.ConfigSecretID.String(),
		ClusterName:     c.Name,
		NodePoolName:    nodePoolName,
		OrganizationID:  c.OrganizationID,

		NodeImage: nodePoolUpdate.Image,
		Version:   nodePoolUpdate.Version,

		Options: pke.NodePoolUpdateOptions{
			MaxSurge:       nodePoolUpdate.Options.MaxSurge,
			MaxBatchSize:   nodePoolUpdate.Options.MaxBatchSize,
			MaxUnavailable: nodePoolUpdate.Options.MaxUnavailable,
			Drain: pke.NodePoolUpdateDrainOptions{
				Timeout:     nodePoolUpdate.Options.Drain.Timeout,
				FailOnError: nodePoolUpdate.Options.Drain.FailOnError,
				PodSelector: nodePoolUpdate.Options.Drain.PodSelector,
			},
		},
		ClusterTags: c.Tags,
	}

	e, err := n.workflowClient.StartWorkflow(ctx, workflowOptions, pkeawsworkflow.UpdateNodePoolWorkflowName, input)
	if err != nil {
		return "", errors.WrapWithDetails(err, "failed to start workflow", "workflow", pkeawsworkflow.UpdateNodePoolWorkflowName)
	}

	return e.ID, nil
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
