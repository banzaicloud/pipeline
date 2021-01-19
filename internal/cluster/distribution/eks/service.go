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

package eks

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"

	"github.com/banzaicloud/pipeline/internal/cluster"
	sdkCloudFormation "github.com/banzaicloud/pipeline/pkg/sdk/providers/amazon/cloudformation"
)

// +testify:mock

// Service provides an interface to EKS clusters.
type Service interface {
	// CreateNodePool creates a new node pool with the specified attributes.
	CreateNodePool(ctx context.Context, clusterID uint, nodePool NewNodePool) (err error)

	// DeleteNodePool deletes an existing node pool.
	DeleteNodePool(ctx context.Context, clusterID uint, nodePoolName string) (isDeleted bool, err error)

	// UpdateCluster updates a cluster.
	//
	// This method accepts a partial body representation.
	UpdateCluster(ctx context.Context, clusterID uint, clusterUpdate ClusterUpdate) error

	// UpdateNodePool updates an existing node pool in a cluster.
	//
	// This method accepts a partial body representation.
	UpdateNodePool(ctx context.Context, clusterID uint, nodePoolName string, nodePoolUpdate NodePoolUpdate) (string, error)

	// ListNodePools lists node pools from a cluster.
	ListNodePools(ctx context.Context, clusterID uint) ([]NodePool, error)
}

// ClusterUpdate describes a cluster update request.
//
// A cluster update contains a partial representation of the cluster resource,
// updating only the changed values.
type ClusterUpdate struct {
	Version string `mapstructure:"version"`
}

// NodePoolUpdate describes a node pool update request.
//
// A node pool update contains a partial representation of the node pool resource,
// updating only the changed values.
type NodePoolUpdate struct {
	VolumeEncryption *NodePoolVolumeEncryption `mapstructure:"volumeEncryption,omitempty"`
	VolumeSize       int                       `mapstructure:"volumeSize"`

	Image            string   `mapstructure:"image"`
	SecurityGroups   []string `mapstructure:"securityGroups"`
	UseInstanceStore *bool    `mapstructure:"useInstanceStore,omitempty"`

	Options NodePoolUpdateOptions `mapstructure:"options"`
}

type NodePoolUpdateOptions struct {

	// Maximum number of extra nodes that can be created during the update.
	MaxSurge int `mapstructure:"maxSurge"`

	// Maximum number of nodes that can be updated simultaneously.
	MaxBatchSize int `mapstructure:"maxBatchSize"`

	// Maximum number of nodes that can be unavailable during the update.
	MaxUnavailable int `mapstructure:"maxUnavailable"`

	// Kubernetes node drain specific options.
	Drain NodePoolUpdateDrainOptions `mapstructure:"drain"`
}

type NodePoolUpdateDrainOptions struct {
	Timeout int `mapstructure:"timeout"`

	FailOnError bool `mapstructure:"failOnError"`

	PodSelector string `mapstructure:"podSelector"`
}

// NodePool encapsulates information about a cluster node pool.
type NodePool struct {
	Name             string                    `mapstructure:"name"`
	Labels           map[string]string         `mapstructure:"labels"`
	Size             int                       `mapstructure:"size"`
	Autoscaling      Autoscaling               `mapstructure:"autoscaling"`
	VolumeEncryption *NodePoolVolumeEncryption `mapstructure:"volumeEncryption,omitempty"`
	VolumeSize       int                       `mapstructure:"volumeSize"`
	InstanceType     string                    `mapstructure:"instanceType"`
	Image            string                    `mapstructure:"image"`
	SpotPrice        string                    `mapstructure:"spotPrice"`
	SecurityGroups   []string                  `mapstructure:"securityGroups,omitempty"`
	SubnetID         string                    `mapstructure:"subnetId"`
	UseInstanceStore bool                      `mapstructure:"UseInstanceStore"`
	Status           NodePoolStatus            `mapstructure:"status"`
	StatusMessage    string                    `mapstructure:"statusMessage"`
}

// NewNodePoolFromCFStack initializes a node pool object from a CloudFormation
// stack.
func NewNodePoolFromCFStack(name string, labels map[string]string, stack *cloudformation.Stack) (nodePool NodePool) {
	var nodePoolParameters struct {
		ClusterAutoscalerEnabled    bool   `mapstructure:"ClusterAutoscalerEnabled"`
		NodeAutoScalingGroupMaxSize int    `mapstructure:"NodeAutoScalingGroupMaxSize"`
		NodeAutoScalingGroupMinSize int    `mapstructure:"NodeAutoScalingGroupMinSize"`
		NodeAutoScalingInitSize     int    `mapstructure:"NodeAutoScalingInitSize"`
		NodeImageID                 string `mapstructure:"NodeImageId"`
		NodeInstanceType            string `mapstructure:"NodeInstanceType"`
		NodeSpotPrice               string `mapstructure:"NodeSpotPrice"`
		NodeVolumeEncryptionEnabled string `mapstructure:"NodeVolumeEncryptionEnabled,omitempty"` // Note: CustomNodeSecurityGroups is only available from template version 2.1.0.
		NodeVolumeEncryptionKeyARN  string `mapstructure:"NodeVolumeEncryptionKeyARN,omitempty"`  // Note: CustomNodeSecurityGroups is only available from template version 2.1.0.
		NodeVolumeSize              int    `mapstructure:"NodeVolumeSize"`
		CustomNodeSecurityGroups    string `mapstructure:"CustomNodeSecurityGroups,omitempty"` // Note: CustomNodeSecurityGroups is only available from template version 2.0.0.
		Subnets                     string `mapstructure:"Subnets"`
		UseInstanceStore            string `mapstructure:"UseInstanceStore,omitempty"`
	}

	err := sdkCloudFormation.ParseStackParameters(stack.Parameters, &nodePoolParameters)
	if err != nil {
		return NewNodePoolWithNoValues(name, NodePoolStatusError, err.Error())
	}

	nodePool.Name = name
	nodePool.Labels = labels
	nodePool.Size = nodePoolParameters.NodeAutoScalingInitSize
	nodePool.Autoscaling = Autoscaling{
		Enabled: nodePoolParameters.ClusterAutoscalerEnabled,
		MinSize: nodePoolParameters.NodeAutoScalingGroupMinSize,
		MaxSize: nodePoolParameters.NodeAutoScalingGroupMaxSize,
	}

	if nodePoolParameters.NodeVolumeEncryptionEnabled != "" {
		nodePool.VolumeEncryption = &NodePoolVolumeEncryption{}
		nodePool.VolumeEncryption.Enabled, err = strconv.ParseBool(nodePoolParameters.NodeVolumeEncryptionEnabled)
		if err != nil {
			return NewNodePoolWithNoValues(name, NodePoolStatusError, "invalid encryption information")
		}

		nodePool.VolumeEncryption.EncryptionKeyARN = nodePoolParameters.NodeVolumeEncryptionKeyARN
	}

	nodePool.VolumeSize = nodePoolParameters.NodeVolumeSize
	nodePool.InstanceType = nodePoolParameters.NodeInstanceType
	nodePool.Image = nodePoolParameters.NodeImageID
	nodePool.SpotPrice = nodePoolParameters.NodeSpotPrice

	if nodePoolParameters.CustomNodeSecurityGroups != "" {
		nodePool.SecurityGroups = strings.Split(nodePoolParameters.CustomNodeSecurityGroups, ",")
		sort.Strings(nodePool.SecurityGroups)
	}

	nodePool.SubnetID = nodePoolParameters.Subnets // Note: currently we ensure a single value at creation.
	nodePool.Status = NewNodePoolStatusFromCFStackStatus(aws.StringValue(stack.StackStatus))
	nodePool.StatusMessage = aws.StringValue(stack.StackStatusReason)

	if nodePoolParameters.UseInstanceStore != "" {
		useInstanceStore, err := strconv.ParseBool(nodePoolParameters.UseInstanceStore)
		if err != nil {
			return NewNodePoolWithNoValues(name, NodePoolStatusError, "invalid useInstanceStore information")
		}
		nodePool.UseInstanceStore = useInstanceStore
	}

	return nodePool
}

// NewNodePoolFromCFStackDescriptionError initializes a node pool with the
// information derived from the CloudFormation stack description error.
func NewNodePoolFromCFStackDescriptionError(err error, existingNodePool ExistingNodePool) (nodePool NodePool) {
	if existingNodePool.StackID == "" &&
		existingNodePool.Status == NodePoolStatusEmpty &&
		existingNodePool.StatusMessage == "" {
		// Note: older node pool with no stored stack ID, status or
		// status message and DescribeStacks() doesn't work with stack
		// name for deleting stacks.
		return NewNodePoolWithNoValues(existingNodePool.Name, NodePoolStatusDeleting, "")
	} else if existingNodePool.StackID == "" &&
		existingNodePool.Status != NodePoolStatusEmpty {
		// Note: node pool is in the database already, but the stack is
		// not existing thus it is either being created, failed
		// creation with error before CloudFormation stack creation
		// would have been started.
		return NewNodePoolWithNoValues(existingNodePool.Name, existingNodePool.Status, existingNodePool.StatusMessage)
	}

	return NewNodePoolWithNoValues( // Note: unexpected failure.
		existingNodePool.Name,
		NodePoolStatusUnknown,
		fmt.Sprintf("retrieving node pool information failed: %s", err),
	)
}

func NewNodePoolWithNoValues(name string, status NodePoolStatus, statusMessage string) (nodePool NodePool) {
	return NodePool{
		Name:          name,
		Status:        status,
		StatusMessage: statusMessage,
	}
}

// NodePoolStatus represents the possible states of a node pool.
type NodePoolStatus string

const (
	// NodePoolStatusCreating is the status used when the node pool resources
	// are being provisioned.
	NodePoolStatusCreating NodePoolStatus = "CREATING"

	// NodePoolStatusDeleted is the status used when the node pool resources
	// are already removed.
	NodePoolStatusDeleted NodePoolStatus = "DELETED"

	// NodePoolStatusDeleting is the status used when the node pool resources
	// are being removed.
	NodePoolStatusDeleting NodePoolStatus = "DELETING"

	// NodePoolStatusEmpty is the status used when the node pool status needs to
	// be explicitly set to an empty value. This is also the type's default
	// value.
	NodePoolStatusEmpty NodePoolStatus = ""

	// NodePoolStatusCreating is the status returned when the node pool
	// is in an invalid state or an operation cannot be performed on it.
	NodePoolStatusError NodePoolStatus = "ERROR"

	// NodePoolStatusCreating is the status returned when the node pool
	// is in a healthy, idle state.
	NodePoolStatusReady NodePoolStatus = "READY"

	// NodePoolStatusUnknown is the status returned when the node pool cannot be
	// examined.
	NodePoolStatusUnknown NodePoolStatus = "UNKNOWN"

	// NodePoolStatusUpdating is the status returned when the node pool
	// resources are being changed.
	NodePoolStatusUpdating NodePoolStatus = "UPDATING"
)

// NewNodePoolStatusFromCFStackStatus translates a CloudFormation stack status
// into a node pool status.
func NewNodePoolStatusFromCFStackStatus(cfStackStatus string) (nodePoolStatus NodePoolStatus) {
	switch {
	case cfStackStatus == cloudformation.StackStatusCreateInProgress:
		return NodePoolStatusCreating
	case cfStackStatus == cloudformation.StackStatusDeleteComplete:
		return NodePoolStatusDeleted // Note: CF stack is deleted, but DB entry is still existing.
	case cfStackStatus == cloudformation.StackStatusDeleteInProgress:
		return NodePoolStatusDeleting
	case strings.HasSuffix(cfStackStatus, "_COMPLETE"):
		return NodePoolStatusReady
	case strings.HasSuffix(cfStackStatus, "_FAILED"):
		return NodePoolStatusError
	case strings.HasSuffix(cfStackStatus, "_IN_PROGRESS"):
		return NodePoolStatusUpdating
	default:
		return NodePoolStatusUnknown
	}
}

// Autoscaling describes the EKS node pool's autoscaling settings.
type Autoscaling struct {
	Enabled bool `mapstructure:"enabled"`
	MinSize int  `mapstructure:"minSize"`
	MaxSize int  `mapstructure:"maxSize"`
}

// NodePoolVolumeEncryption describes the EKS node pool encryption details.
type NodePoolVolumeEncryption struct {
	Enabled          bool   `mapstructure:"enabled"`
	EncryptionKeyARN string `mapstructure:"encryptionKeyARN"`
}

// NewService returns a new Service instance.
func NewService(
	genericClusters Store,
	clusterManager ClusterManager,
	nodePools NodePoolStore,
	nodePoolManager NodePoolManager,
	nodePoolProcessor NodePoolProcessor,
	nodePoolValidator NodePoolValidator,
) Service {
	return service{
		genericClusters:   genericClusters,
		clusterManager:    clusterManager,
		nodePools:         nodePools,
		nodePoolManager:   nodePoolManager,
		nodePoolProcessor: nodePoolProcessor,
		nodePoolValidator: nodePoolValidator,
	}
}

type service struct {
	genericClusters   Store
	clusterManager    ClusterManager
	nodePools         NodePoolStore
	nodePoolManager   NodePoolManager
	nodePoolProcessor NodePoolProcessor
	nodePoolValidator NodePoolValidator
}

// +testify:mock:testOnly=true

// NodePoolManager is responsible for managing node pools.
type NodePoolManager interface {
	// CreateNodePool creates a new node pool in a cluster with the specified
	// attributes.
	CreateNodePool(ctx context.Context, c cluster.Cluster, nodePool NewNodePool) (err error)

	// DeleteNodePool deletes an existing node pool in a cluster.
	DeleteNodePool(
		ctx context.Context, c cluster.Cluster, existingNodePool ExistingNodePool, shouldUpdateClusterStatus bool,
	) (err error)

	// UpdateNodePool updates an existing node pool in a cluster.
	UpdateNodePool(ctx context.Context, c cluster.Cluster, nodePoolName string, nodePoolUpdate NodePoolUpdate) (string, error)

	// ListNodePools lists node pools from a cluster.
	ListNodePools(
		ctx context.Context,
		c cluster.Cluster,
		existingNodePools map[string]ExistingNodePool,
	) ([]NodePool, error)
}

// ClusterManager is responsible for managing clusters.
type ClusterManager interface {
	// UpdateCluster updates an existing cluster.
	UpdateCluster(ctx context.Context, c cluster.Cluster, clusterUpdate ClusterUpdate) error
}

// CreateNodePool creates a new node pool in a cluster with the specified
// attributes.
//
// Implements the Service interface.
func (s service) CreateNodePool(ctx context.Context, clusterID uint, nodePool NewNodePool) (err error) {
	c, err := s.genericClusters.GetCluster(ctx, clusterID)
	if err != nil {
		return err
	}

	if err := s.nodePoolValidator.ValidateNewNodePool(ctx, c, nodePool); err != nil {
		return err
	}

	nodePool, err = s.nodePoolProcessor.ProcessNewNodePool(ctx, c, nodePool)
	if err != nil {
		return err
	}

	err = s.genericClusters.SetStatus(ctx, clusterID, cluster.Updating, "creating node pool")
	if err != nil {
		return err
	}

	err = s.nodePoolManager.CreateNodePool(ctx, c, nodePool)
	if err != nil {
		return err
	}

	return nil
}

func (s service) DeleteNodePool(ctx context.Context, clusterID uint, nodePoolName string) (isDeleted bool, err error) {
	c, err := s.genericClusters.GetCluster(ctx, clusterID)
	if err != nil {
		return false, err
	}

	existingNodePools, err := s.nodePools.ListNodePools(ctx, c.OrganizationID, c.ID, c.Name)
	if err != nil {
		return false, err
	}

	existingNodePool, isExisting := existingNodePools[nodePoolName]
	if !isExisting {
		return true, nil
	}

	err = s.genericClusters.SetStatus(ctx, clusterID, cluster.Updating, "deleting node pool")
	if err != nil {
		return false, err
	}

	err = s.nodePoolManager.DeleteNodePool(ctx, c, existingNodePool, true)
	if err != nil {
		return false, err
	}

	return false, nil
}

func (s service) UpdateCluster(
	ctx context.Context,
	clusterID uint,
	clusterUpdate ClusterUpdate,
) error {
	c, err := s.genericClusters.GetCluster(ctx, clusterID)
	if err != nil {
		return err
	}

	err = s.genericClusters.SetStatus(ctx, clusterID, cluster.Updating, "updating cluster")
	if err != nil {
		return err
	}

	return s.clusterManager.UpdateCluster(ctx, c, clusterUpdate)
}

func (s service) UpdateNodePool(
	ctx context.Context,
	clusterID uint,
	nodePoolName string,
	nodePoolUpdate NodePoolUpdate,
) (string, error) {
	// TODO: check if node pool exists

	c, err := s.genericClusters.GetCluster(ctx, clusterID)
	if err != nil {
		return "", err
	}

	err = s.genericClusters.SetStatus(ctx, clusterID, cluster.Updating, "updating node pool")
	if err != nil {
		return "", err
	}

	return s.nodePoolManager.UpdateNodePool(ctx, c, nodePoolName, nodePoolUpdate)
}

// ListNodePools lists node pools from a cluster.
func (s service) ListNodePools(ctx context.Context, clusterID uint) ([]NodePool, error) {
	c, err := s.genericClusters.GetCluster(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	existingNodePools, err := s.nodePools.ListNodePools(ctx, c.OrganizationID, c.ID, c.Name)
	if err != nil {
		return nil, err
	}

	return s.nodePoolManager.ListNodePools(ctx, c, existingNodePools)
}

// +testify:mock:testOnly=true

// Store provides an interface to the generic Cluster model persistence.
type Store interface {
	// GetCluster returns a generic Cluster.
	// Returns an error with the NotFound behavior when the cluster cannot be found.
	GetCluster(ctx context.Context, id uint) (cluster.Cluster, error)

	// SetStatus sets the cluster status.
	SetStatus(ctx context.Context, id uint, status string, statusMessage string) error
}
