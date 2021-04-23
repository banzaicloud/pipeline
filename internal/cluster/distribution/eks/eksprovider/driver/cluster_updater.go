// Copyright © 2019 Banzai Cloud
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

package driver

import (
	"context"
	"fmt"
	"sort"
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/sirupsen/logrus"
	"go.uber.org/cadence/client"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	pkgEks "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/ekscluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksmodel"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/src/cluster"
)

type EksClusterUpdater struct {
	logger         logrus.FieldLogger
	workflowClient client.Client
}

type updateValidationError struct {
	msg string

	invalidRequest     bool
	preconditionFailed bool
}

func (e *updateValidationError) Error() string {
	return e.msg
}

func (e *updateValidationError) IsInvalid() bool {
	return e.invalidRequest
}

func (e *updateValidationError) IsPreconditionFailed() bool {
	return e.preconditionFailed
}

func NewEksClusterUpdater(logger logrus.FieldLogger, workflowClient client.Client) EksClusterUpdater {
	return EksClusterUpdater{
		logger:         logger,
		workflowClient: workflowClient,
	}
}

func (c *EksClusterUpdater) validate(ctx context.Context, eksCluster *cluster.EKSCluster) error {
	status, err := eksCluster.GetStatus()
	if err != nil {
		return errors.Wrap(err, "could not get Cluster status")
	}

	if status.Status != pkgCluster.Running && status.Status != pkgCluster.Warning {
		return errors.WithDetails(
			&updateValidationError{
				msg:                fmt.Sprintf("Cluster is not in %s or %s state yet", pkgCluster.Running, pkgCluster.Warning),
				preconditionFailed: true,
			},
			"status", status.Status,
		)
	}

	return nil
}

func (c *EksClusterUpdater) prepare(ctx context.Context, eksCluster *cluster.EKSCluster, request *pkgCluster.UpdateClusterRequest) error {
	eksCluster.AddDefaultsToUpdate(request)

	if err := eksCluster.CheckEqualityToUpdate(request); err != nil {
		return &updateValidationError{
			msg:            err.Error(),
			invalidRequest: true,
		}
	}

	if err := request.Validate(); err != nil {
		return &updateValidationError{
			msg:            err.Error(),
			invalidRequest: true,
		}
	}

	return nil
}

func (c *EksClusterUpdater) update(ctx context.Context, logger logrus.FieldLogger, eksCluster *cluster.EKSCluster, request *pkgCluster.UpdateClusterRequest, userID uint) error {
	logger.Info("start EKS Cluster update flow")

	if err := eksCluster.SetStatus(pkgCluster.Updating, pkgCluster.UpdatingMessage); err != nil {
		return errors.WrapIf(err, "could not update cluster status")
	}

	modelCluster := eksCluster.GetModel()

	requestedDeletedNodePools, requestedNewNodePools, requestedUpdatedNodePools, err :=
		newNodePoolsFromUpdateRequest(modelCluster.NodePools, request.EKS.NodePools)
	if err != nil {
		return err
	}

	clusterSubnets, err := newClusterUpdateSubnetsFromModels(modelCluster.Subnets)
	if err != nil {
		return err
	}

	newNodePoolSubnetIDs, err := newNodePoolSubnetIDsFromRequestedNewNodePools(requestedNewNodePools, clusterSubnets)
	if err != nil {
		return err
	}

	nodePoolLabels, err := newNodePoolLabels(
		ctx,
		eksCluster,
		requestedDeletedNodePools,
		requestedNewNodePools,
		requestedUpdatedNodePools,
	)
	if err != nil {
		return err
	}

	newNodePools, err := newNodePoolsFromRequestedNewNodePools(requestedNewNodePools, newNodePoolSubnetIDs)
	if err != nil {
		return err
	}

	input := cluster.EKSUpdateClusterstructureWorkflowInput{
		Region:                 eksCluster.GetLocation(),
		OrganizationID:         eksCluster.GetOrganizationId(),
		SecretID:               eksCluster.GetSecretId(),
		ConfigSecretID:         eksCluster.GetConfigSecretId(),
		ClusterID:              eksCluster.GetID(),
		ClusterName:            eksCluster.GetName(),
		Tags:                   modelCluster.Cluster.Tags,
		UpdaterUserID:          userID,
		DeletableNodePoolNames: newNodePoolNamesFromRequestedDeletedNodePools(requestedDeletedNodePools),
		NewNodePools:           newNodePools,
		NewNodePoolSubnetIDs:   newNodePoolSubnetIDs,
		NodePoolLabels:         nodePoolLabels,
		UpdatedNodePools:       newASGsFromRequestedUpdatedNodePools(requestedUpdatedNodePools, modelCluster.NodePools),
	}

	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: 1 * 24 * time.Hour,
	}
	exec, err := c.workflowClient.ExecuteWorkflow(ctx, workflowOptions, cluster.EKSUpdateClusterWorkflowName, input)
	if err != nil {
		return err
	}

	err = eksCluster.SetCurrentWorkflowID(exec.GetID())
	if err != nil {
		return err
	}

	return nil
}

func (c *EksClusterUpdater) UpdateCluster(ctx context.Context,
	request *pkgCluster.UpdateClusterRequest,
	commonCluster cluster.CommonCluster,
	userID uint) error {
	eksCluster := commonCluster.(*cluster.EKSCluster)

	logger := c.logger.WithFields(logrus.Fields{
		"clusterName":    eksCluster.GetName(),
		"clusterID":      eksCluster.GetID(),
		"organizationID": eksCluster.GetOrganizationId(),
	})
	logger.Info("start deleting EKS Cluster")

	logger.Debug("validate update request")
	err := c.validate(ctx, eksCluster)
	if err != nil {
		return errors.WithMessage(err, "cluster update validation failed")
	}

	err = c.prepare(ctx, eksCluster, request)
	if err != nil {
		return errors.WithMessage(err, "could not prepare cluster")
	}

	err = c.update(ctx, logger, eksCluster, request, userID)
	if err != nil {
		return errors.WrapIf(err, "error updating cluster")
	}

	return nil
}

func newASGsFromRequestedUpdatedNodePools(
	requestedUpdatedNodePools map[string]*pkgEks.NodePool,
	currentNodePools []*eksmodel.AmazonNodePoolsModel,
) []workflow.AutoscaleGroup {
	updatedNodePools := make([]workflow.AutoscaleGroup, 0, len(requestedUpdatedNodePools))

	creators := make(map[string]uint, len(requestedUpdatedNodePools))
	for _, currentNodePool := range currentNodePools {
		creators[currentNodePool.Name] = currentNodePool.CreatedBy
	}

	for nodePoolName, nodePool := range requestedUpdatedNodePools {
		var volumeEncryption *eks.NodePoolVolumeEncryption
		if nodePool.VolumeEncryption != nil {
			volumeEncryption = &eks.NodePoolVolumeEncryption{
				Enabled:          nodePool.VolumeEncryption.Enabled,
				EncryptionKeyARN: nodePool.VolumeEncryption.EncryptionKeyARN,
			}
		}

		updatedNodePools = append(updatedNodePools, workflow.AutoscaleGroup{
			Name:                 nodePoolName,
			NodeSpotPrice:        nodePool.SpotPrice,
			Autoscaling:          nodePool.Autoscaling,
			NodeMinCount:         nodePool.MinCount,
			NodeMaxCount:         nodePool.MaxCount,
			Count:                nodePool.Count,
			NodeVolumeEncryption: volumeEncryption,
			NodeVolumeSize:       nodePool.VolumeSize,
			NodeImage:            nodePool.Image,
			NodeInstanceType:     nodePool.InstanceType,
			SecurityGroups:       nodePool.SecurityGroups,
			UseInstanceStore:     nodePool.UseInstanceStore,
			Labels:               nodePool.Labels,
			Delete:               false,
			Create:               false,
			CreatedBy:            creators[nodePoolName],
		})
	}

	sort.Slice(updatedNodePools, func(first, second int) (isLessThan bool) {
		return updatedNodePools[first].Name < updatedNodePools[second].Name
	})

	return updatedNodePools
}

// newClusterUpdateSubnetsFromModels returns the collection of clusters subnets
// transformed from the specified subnet models or alternatively the occurring
// error.
func newClusterUpdateSubnetsFromModels(clusterSubnetModels []*eksmodel.EKSSubnetModel) ([]workflow.Subnet, error) {
	clusterSubnets := make([]workflow.Subnet, 0, len(clusterSubnetModels))
	clusterSubnetErrors := make([]error, 0, len(clusterSubnetModels))

	for _, subnet := range clusterSubnetModels {
		if aws.StringValue(subnet.SubnetId) == "" {
			clusterSubnetErrors = append(
				clusterSubnetErrors,
				errors.Errorf(
					"cluster subnet CIDR %s lacks an ID and subnet creation is not supported during cluster update",
					aws.StringValue(subnet.Cidr),
				),
			)

			continue
		}

		clusterSubnets = append(clusterSubnets, workflow.Subnet{
			SubnetID:         aws.StringValue(subnet.SubnetId),
			Cidr:             aws.StringValue(subnet.Cidr),
			AvailabilityZone: aws.StringValue(subnet.AvailabilityZone),
		})
	}

	if len(clusterSubnetErrors) != 0 {
		return nil, errors.Combine(clusterSubnetErrors...)
	}

	if len(clusterSubnets) == 0 {
		return nil, errors.Errorf("no cluster subnet is available")
	}

	return clusterSubnets, nil
}

// newNodePoolNamesFromRequestedDeletedNodePools returns the collection of the
// names of the specified node pools requested to be deleted.
func newNodePoolNamesFromRequestedDeletedNodePools(nodePoolModels map[string]*eksmodel.AmazonNodePoolsModel) []string {
	nodePoolNames := make([]string, 0, len(nodePoolModels))
	for nodePoolName := range nodePoolModels {
		nodePoolNames = append(nodePoolNames, nodePoolName)
	}

	sort.Slice(nodePoolNames, func(first, second int) (isLessThan bool) {
		return nodePoolNames[first] < nodePoolNames[second]
	})

	return nodePoolNames
}

// newNodePoolLabels returns the determined node pool labels for all the node
// pools being updated by the cluster update.
//
// TODO: remove when UpdateNodePoolWorkflow is refactored and node pool labels
// are passed and synced implicitly in the
// Create-/Delete-/UpdateNodePoolWorkflow operations.
func newNodePoolLabels(
	ctx context.Context,
	eksCluster cluster.CommonCluster,
	requestedDeletedNodePools map[string]*eksmodel.AmazonNodePoolsModel,
	requestedNewNodePools map[string]*pkgEks.NodePool,
	requestedUpdatedNodePools map[string]*pkgEks.NodePool,
) (map[string]map[string]string, error) {
	combinedModifiedNodePoolCount := len(requestedDeletedNodePools) +
		len(requestedNewNodePools) +
		len(requestedUpdatedNodePools)

	clusterNodePoolLabels := make([]cluster.NodePoolLabels, 0, combinedModifiedNodePoolCount)

	for nodePoolName, nodePool := range requestedDeletedNodePools {
		clusterNodePoolLabels = append(clusterNodePoolLabels, cluster.NodePoolLabels{
			NodePoolName: nodePoolName,
			Existing:     true,
			InstanceType: nodePool.NodeInstanceType,
			SpotPrice:    nodePool.NodeSpotPrice,
			// Preemptible:  , // Note: parsed from SpotPrice if specified, defaulted 0.0.
			CustomLabels: nodePool.Labels,
		})
	}

	for nodePoolName, nodePool := range requestedNewNodePools {
		clusterNodePoolLabels = append(clusterNodePoolLabels, cluster.NodePoolLabels{
			NodePoolName: nodePoolName,
			Existing:     false,
			InstanceType: nodePool.InstanceType,
			SpotPrice:    nodePool.SpotPrice,
			// Preemptible:  , // Note: parsed from SpotPrice if specified, defaulted 0.0.
			CustomLabels: nodePool.Labels,
		})
	}

	for nodePoolName, nodePool := range requestedUpdatedNodePools {
		clusterNodePoolLabels = append(clusterNodePoolLabels, cluster.NodePoolLabels{
			NodePoolName: nodePoolName,
			Existing:     true,
			InstanceType: nodePool.InstanceType,
			SpotPrice:    nodePool.SpotPrice,
			// Preemptible:  , // Note: parsed from SpotPrice if specified, defaulted 0.0.
			CustomLabels: nodePool.Labels,
		})
	}

	nodePoolLabels, err := cluster.GetDesiredLabelsForCluster(ctx, eksCluster, clusterNodePoolLabels)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get desired labels for cluster")
	}

	return nodePoolLabels, nil
}

// newNodePoolsFromRequest returns the requested node pool deletions, creations,
// updates based on the current and requested node pools.
func newNodePoolsFromUpdateRequest(
	currentNodePools []*eksmodel.AmazonNodePoolsModel,
	requestedNodePools map[string]*pkgEks.NodePool,
) (
	requestedDeletedNodePools map[string]*eksmodel.AmazonNodePoolsModel,
	requestedNewNodePools map[string]*pkgEks.NodePool,
	requestedUpdatedNodePools map[string]*pkgEks.NodePool,
	err error,
) {
	existingNodePools := make(map[string]bool, len(currentNodePools))
	requestedDeletedNodePools = make(map[string]*eksmodel.AmazonNodePoolsModel, len(currentNodePools))
	for _, currentNodePool := range currentNodePools {
		existingNodePools[currentNodePool.Name] = true

		if _, isExisting := requestedNodePools[currentNodePool.Name]; !isExisting {
			requestedDeletedNodePools[currentNodePool.Name] = currentNodePool
		}
	}

	requestedNewNodePools = make(map[string]*pkgEks.NodePool, len(requestedNodePools))
	requestedUpdatedNodePools = make(map[string]*pkgEks.NodePool, len(requestedNodePools))
	for nodePoolName, nodePool := range requestedNodePools {
		if existingNodePools[nodePoolName] {
			requestedUpdatedNodePools[nodePoolName] = nodePool
		} else {
			if len(nodePool.InstanceType) == 0 {
				return nil, nil, nil, pkgErrors.ErrorInstancetypeFieldIsEmpty
			}

			if len(nodePool.Image) == 0 {
				return nil, nil, nil, pkgErrors.ErrorAmazonImageFieldIsEmpty
			}

			if len(nodePool.SpotPrice) == 0 {
				nodePool.SpotPrice = eks.DefaultSpotPrice
			}

			requestedNewNodePools[nodePoolName] = nodePool
		}
	}

	return requestedDeletedNodePools, requestedNewNodePools, requestedUpdatedNodePools, nil
}

// newNodePoolsFromRequestedNewNodePools returns a collection of new node pool
// descriptors for node pool creation based on the specified requested new node
// pools.
func newNodePoolsFromRequestedNewNodePools(
	requestedNewNodePools map[string]*pkgEks.NodePool,
	newNodePoolSubnetIDs map[string][]string,
) ([]eks.NewNodePool, error) {
	if newNodePoolSubnetIDs == nil {
		return nil, errors.New("nil new subnet ID map")
	}

	newNodePools := make([]eks.NewNodePool, 0, len(requestedNewNodePools))
	newNodePoolErrors := make([]error, 0, len(requestedNewNodePools))

	for nodePoolName, nodePool := range requestedNewNodePools {
		if len(newNodePoolSubnetIDs[nodePoolName]) == 0 {
			newNodePoolErrors = append(
				newNodePoolErrors,
				errors.Errorf("no subnet ID specified for node pool %s", nodePoolName),
			)

			continue
		}

		var volumeEncryption *eks.NodePoolVolumeEncryption
		if nodePool.VolumeEncryption != nil {
			volumeEncryption = &eks.NodePoolVolumeEncryption{
				Enabled:          nodePool.VolumeEncryption.Enabled,
				EncryptionKeyARN: nodePool.VolumeEncryption.EncryptionKeyARN,
			}
		}

		newNodePools = append(newNodePools, eks.NewNodePool{
			Name:   nodePoolName,
			Labels: nodePool.Labels,
			Size:   nodePool.Count,
			Autoscaling: eks.Autoscaling{
				Enabled: nodePool.Autoscaling,
				MinSize: nodePool.MinCount,
				MaxSize: nodePool.MaxCount,
			},
			VolumeEncryption: volumeEncryption,
			VolumeSize:       nodePool.VolumeSize,
			InstanceType:     nodePool.InstanceType,
			Image:            nodePool.Image,
			SpotPrice:        nodePool.SpotPrice,
			SecurityGroups:   nodePool.SecurityGroups,
			SubnetID:         newNodePoolSubnetIDs[nodePoolName][0],
			UseInstanceStore: nodePool.UseInstanceStore,
		})
	}

	if len(newNodePoolErrors) != 0 {
		return nil, errors.Combine(newNodePoolErrors...)
	}

	sort.Slice(newNodePools, func(first, second int) (isLessThan bool) {
		return newNodePools[first].Name < newNodePools[second].Name
	})

	return newNodePools, nil
}

// newNodePoolSubnetIDsFromRequestedNewNodePools returns the matched cluster
// subnet IDs for the requested new node pools subnet request based on ID or
// CIDR match or alternatively the occurring error.
func newNodePoolSubnetIDsFromRequestedNewNodePools(
	requestedNewNodePools map[string]*pkgEks.NodePool,
	clusterSubnets []workflow.Subnet,
) (map[string][]string, error) {
	if len(clusterSubnets) == 0 {
		return nil, errors.New("empty cluster subnet list")
	}

	nodePoolSubnetIDs := make(map[string][]string, len(requestedNewNodePools))
	nodePoolSubnetErrors := make([]error, 0, len(requestedNewNodePools))

	for nodePoolName, nodePool := range requestedNewNodePools {
		if nodePool.Subnet == nil {
			nodePoolSubnetIDs[nodePoolName] = append(nodePoolSubnetIDs[nodePoolName], clusterSubnets[0].SubnetID)
		} else if nodePool.Subnet.SubnetId != "" ||
			nodePool.Subnet.Cidr != "" {
			for _, clusterSubnet := range clusterSubnets {
				if clusterSubnet.SubnetID == nodePool.Subnet.SubnetId ||
					(nodePool.Subnet.SubnetId == "" && clusterSubnet.Cidr == nodePool.Subnet.Cidr) {
					// Note: new cluster subnets won't be created at cluster update.
					nodePoolSubnetIDs[nodePoolName] = append(nodePoolSubnetIDs[nodePoolName], clusterSubnet.SubnetID)
				}
			}
		} else {
			nodePoolSubnetErrors = append(
				nodePoolSubnetErrors,
				errors.Errorf("node pool %s is missing both subnet ID and CIDR: %+v", nodePoolName, nodePool.Subnet),
			)

			continue
		}

		if len(nodePoolSubnetIDs[nodePoolName]) == 0 ||
			nodePoolSubnetIDs[nodePoolName][0] == "" { // Note: new cluster subnets won't be created at cluster update.
			nodePoolSubnetErrors = append(
				nodePoolSubnetErrors,
				errors.Errorf("subnet ID not found for node pool %s with subnet %+v", nodePoolName, nodePool.Subnet),
			)

			continue
		}
	}

	if len(nodePoolSubnetErrors) != 0 {
		return nil, errors.Combine(nodePoolSubnetErrors...)
	}

	return nodePoolSubnetIDs, nil
}
