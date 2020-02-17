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
	"reflect"
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/sirupsen/logrus"
	"go.uber.org/cadence/client"

	"github.com/banzaicloud/pipeline/internal/providers/amazon/eks/workflow"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgEks "github.com/banzaicloud/pipeline/pkg/cluster/eks"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/src/cluster"
	"github.com/banzaicloud/pipeline/src/model"
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

func createNodePoolsFromUpdateRequest(eksCluster *cluster.EKSCluster, requestedNodePools map[string]*pkgEks.NodePool, userId uint) ([]*model.AmazonNodePoolsModel, error) {

	currentNodePoolMap := make(map[string]*model.AmazonNodePoolsModel, len(eksCluster.GetModel().EKS.NodePools))
	for _, nodePool := range eksCluster.GetModel().EKS.NodePools {
		currentNodePoolMap[nodePool.Name] = nodePool
	}

	updatedNodePools := make([]*model.AmazonNodePoolsModel, 0, len(requestedNodePools))

	for nodePoolName, nodePool := range requestedNodePools {
		if currentNodePoolMap[nodePoolName] != nil {
			// update existing node pool
			updatedNodePools = append(updatedNodePools, &model.AmazonNodePoolsModel{
				ID:               currentNodePoolMap[nodePoolName].ID,
				CreatedBy:        currentNodePoolMap[nodePoolName].CreatedBy,
				CreatedAt:        currentNodePoolMap[nodePoolName].CreatedAt,
				ClusterID:        currentNodePoolMap[nodePoolName].ClusterID,
				Name:             nodePoolName,
				NodeInstanceType: currentNodePoolMap[nodePoolName].NodeInstanceType,
				NodeImage:        currentNodePoolMap[nodePoolName].NodeImage,
				NodeSpotPrice:    currentNodePoolMap[nodePoolName].NodeSpotPrice,
				Autoscaling:      nodePool.Autoscaling,
				NodeMinCount:     nodePool.MinCount,
				NodeMaxCount:     nodePool.MaxCount,
				Count:            nodePool.Count,
				Labels:           nodePool.Labels,
				Delete:           false,
			})

		} else {
			// new node pool

			// ---- [ Node instanceType check ] ---- //
			if len(nodePool.InstanceType) == 0 {
				// c.log.Errorf("instanceType is missing for nodePool %v", nodePoolName)
				return nil, pkgErrors.ErrorInstancetypeFieldIsEmpty
			}

			// ---- [ Node image check ] ---- //
			if len(nodePool.Image) == 0 {
				// c.log.Errorf("image is missing for nodePool %v", nodePoolName)
				return nil, pkgErrors.ErrorAmazonImageFieldIsEmpty
			}

			// ---- [ Node spot price ] ---- //
			if len(nodePool.SpotPrice) == 0 {
				nodePool.SpotPrice = pkgEks.DefaultSpotPrice
			}

			updatedNodePools = append(updatedNodePools, &model.AmazonNodePoolsModel{
				CreatedBy:        userId,
				Name:             nodePoolName,
				NodeInstanceType: nodePool.InstanceType,
				NodeImage:        nodePool.Image,
				NodeSpotPrice:    nodePool.SpotPrice,
				Autoscaling:      nodePool.Autoscaling,
				NodeMinCount:     nodePool.MinCount,
				NodeMaxCount:     nodePool.MaxCount,
				Count:            nodePool.Count,
				Delete:           false,
				Labels:           nodePool.Labels,
			})
		}
	}

	for _, nodePool := range eksCluster.GetModel().EKS.NodePools {
		if requestedNodePools[nodePool.Name] == nil {
			updatedNodePools = append(updatedNodePools, &model.AmazonNodePoolsModel{
				ID:        nodePool.ID,
				ClusterID: nodePool.ClusterID,
				Name:      nodePool.Name,
				Labels:    nodePool.Labels,
				CreatedAt: nodePool.CreatedAt,
				Delete:    true,
			})
		}
	}
	return updatedNodePools, nil
}

// isDifferent compares x and y interfaces with deep equal
func isDifferent(x interface{}, y interface{}) error {
	if reflect.DeepEqual(x, y) {
		return pkgErrors.ErrorNotDifferentInterfaces
	}

	return nil
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

	scaleOptionsChanged := isDifferent(request.ScaleOptions, eksCluster.GetScaleOptions()) == nil
	if scaleOptionsChanged {
		eksCluster.SetScaleOptions(request.ScaleOptions)
	}

	clusterPropertiesChanged := true
	if err := eksCluster.CheckEqualityToUpdate(request); err != nil {
		clusterPropertiesChanged = false
		if !scaleOptionsChanged {
			return &updateValidationError{
				msg:            err.Error(),
				invalidRequest: true,
			}
		}
	}

	if !clusterPropertiesChanged && !scaleOptionsChanged {
		return nil
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

	modelNodePools, err := createNodePoolsFromUpdateRequest(eksCluster, request.EKS.NodePools, userID)
	if err != nil {
		return err
	}

	var nodePoolLabelMap map[string]map[string]string
	{

		nodePoolLabels := make([]cluster.NodePoolLabels, 0)
		for _, np := range modelNodePools {
			nodePoolLabels = append(nodePoolLabels, cluster.NodePoolLabels{
				NodePoolName: np.Name,
				Existing:     np.ID != 0,
				InstanceType: np.NodeInstanceType,
				CustomLabels: np.Labels,
				SpotPrice:    np.NodeSpotPrice,
			})
		}

		nodePoolLabelMap, err = cluster.GetDesiredLabelsForCluster(ctx, eksCluster, nodePoolLabels)
		if err != nil {
			return errors.WrapIf(err, "failed to get desired labels for cluster")
		}
	}

	modelCluster := eksCluster.GetEKSModel()

	subnets := make([]workflow.Subnet, 0)
	for _, subnet := range modelCluster.Subnets {
		subnets = append(subnets, workflow.Subnet{
			SubnetID:         aws.StringValue(subnet.SubnetId),
			Cidr:             aws.StringValue(subnet.Cidr),
			AvailabilityZone: aws.StringValue(subnet.AvailabilityZone),
		})
	}

	subnetMapping := make(map[string][]workflow.Subnet)
	for _, nodePool := range modelNodePools {
		// set subnets only for node pools to be updated
		if nodePool.Delete || nodePool.ID != 0 {
			continue
		}
		for reqNodePoolName, reqNodePool := range request.EKS.NodePools {
			if reqNodePoolName == nodePool.Name {
				if reqNodePool.Subnet == nil {
					logger.WithField("nodePool", nodePool.Name).Info("no subnet specified for node pool in the update Request")
					subnetMapping[nodePool.Name] = append(subnetMapping[nodePool.Name], subnets[0])
				} else {
					for _, subnet := range subnets {
						if (reqNodePool.Subnet.SubnetId != "" && subnet.SubnetID == reqNodePool.Subnet.SubnetId) ||
							(reqNodePool.Subnet.Cidr != "" && subnet.Cidr == reqNodePool.Subnet.Cidr) {
							subnetMapping[nodePool.Name] = append(subnetMapping[nodePool.Name], subnet)
						}
					}
				}
			}
		}
	}

	input := cluster.EKSUpdateClusterstructureWorkflowInput{
		Region:             eksCluster.GetLocation(),
		OrganizationID:     eksCluster.GetOrganizationId(),
		SecretID:           eksCluster.GetSecretId(),
		ConfigSecretID:     eksCluster.GetConfigSecretId(),
		ClusterID:          eksCluster.GetID(),
		ClusterUID:         eksCluster.GetUID(),
		ClusterName:        eksCluster.GetName(),
		ScaleEnabled:       eksCluster.GetScaleOptions() != nil && eksCluster.GetScaleOptions().Enabled,
		NodeInstanceRoleID: modelCluster.NodeInstanceRoleId,
		NodePoolLabels:     nodePoolLabelMap,
		GenerateSSH:        eksCluster.IsSSHGenerated(),
	}

	input.Subnets = subnets
	input.ASGSubnetMapping = subnetMapping

	asgList := make([]workflow.AutoscaleGroup, 0)
	for _, np := range modelNodePools {
		asg := workflow.AutoscaleGroup{
			Name:             np.Name,
			NodeSpotPrice:    np.NodeSpotPrice,
			Autoscaling:      np.Autoscaling,
			NodeMinCount:     np.NodeMinCount,
			NodeMaxCount:     np.NodeMaxCount,
			Count:            np.Count,
			NodeImage:        np.NodeImage,
			NodeInstanceType: np.NodeInstanceType,
			Labels:           np.Labels,
			Delete:           np.Delete,
			CreatedBy:        np.CreatedBy,
		}
		if np.ID == 0 {
			asg.Create = true
		}
		asgList = append(asgList, asg)
	}

	input.AsgList = asgList

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
