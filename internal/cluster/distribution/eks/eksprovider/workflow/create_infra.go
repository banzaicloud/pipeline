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

package workflow

import (
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	pkgCadence "github.com/banzaicloud/pipeline/pkg/cadence"
)

const CreateInfraWorkflowName = "eks-create-infra"

// CreateInfrastructureWorkflowInput holds data needed by the create EKS cluster infrastructure workflow
type CreateInfrastructureWorkflowInput struct {
	Region         string
	OrganizationID uint
	SecretID       string
	SSHSecretID    string

	ClusterUID    string
	ClusterID     uint
	ClusterName   string
	CreatorUserID uint
	VpcID         string
	RouteTableID  string
	VpcCidr       string
	Tags          map[string]string

	Subnets []Subnet

	DefaultUser        bool
	ClusterRoleID      string
	NodeInstanceRoleID string

	KubernetesVersion     string
	EncryptionConfig      []EncryptionConfig
	EndpointPrivateAccess bool
	EndpointPublicAccess  bool

	LogTypes        []string
	NodePools       []eks.NewNodePool
	NodePoolSubnets map[string][]Subnet

	UseGeneratedSSHKey bool

	AuthConfigMap string
}

type CreateInfrastructureWorkflowOutput struct {
	VpcID              string
	NodeInstanceRoleID string
	Subnets            []Subnet
	ConfigSecretID     string
}

type CreateInfrastructureWorkflow struct {
	nodePoolStore eks.NodePoolStore
}

func NewCreateInfrastructureWorkflow(nodePoolStore eks.NodePoolStore) (createInfrastructureWorkflow *CreateInfrastructureWorkflow) {
	return &CreateInfrastructureWorkflow{
		nodePoolStore: nodePoolStore,
	}
}

// Execute executes the Cadence workflow responsible for creating EKS
// cluster infrastructure such as VPC, subnets, EKS master nodes, worker nodes, etc
func (w CreateInfrastructureWorkflow) Execute(ctx workflow.Context, input CreateInfrastructureWorkflowInput) (*CreateInfrastructureWorkflowOutput, error) {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		WaitForCancellation:    true,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          2 * time.Second,
			BackoffCoefficient:       1.5,
			MaximumInterval:          30 * time.Second,
			MaximumAttempts:          5,
			NonRetriableErrorReasons: []string{"cadenceInternal:Panic", ErrReasonStackFailed},
		},
	}

	aoWithHeartbeat := workflow.ActivityOptions{
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		WaitForCancellation:    true,
		HeartbeatTimeout:       45 * time.Second,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          2 * time.Second,
			BackoffCoefficient:       1.5,
			MaximumInterval:          30 * time.Second,
			MaximumAttempts:          5,
			NonRetriableErrorReasons: []string{"cadenceInternal:Panic", ErrReasonStackFailed},
		},
	}

	commonActivityInput := EKSActivityInput{
		OrganizationID: input.OrganizationID,
		SecretID:       input.SecretID,
		Region:         input.Region,
		ClusterName:    input.ClusterName,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	// create IAM role validate activity
	if input.ClusterRoleID != "" {
		{
			activityInput := &ValidateIAMRoleActivityInput{
				EKSActivityInput: commonActivityInput,
				ClusterRoleID:    input.ClusterRoleID,
			}
			validateIAMRoleActivityOutput := ValidateIAMRoleActivityOutput{}
			if err := workflow.ExecuteActivity(ctx, ValidateIAMRoleActivityName, activityInput).Get(ctx, &validateIAMRoleActivityOutput); err != nil {
				return nil, err
			}
		}
	}

	// create IAM roles activity
	var iamRolesCreateActivityFuture workflow.Future
	{
		activityInput := &CreateIamRolesActivityInput{
			EKSActivityInput:   commonActivityInput,
			StackName:          generateStackNameForIam(input.ClusterName),
			DefaultUser:        input.DefaultUser,
			ClusterRoleID:      input.ClusterRoleID,
			NodeInstanceRoleID: input.NodeInstanceRoleID,
			Tags:               input.Tags,
		}
		ctx := workflow.WithActivityOptions(ctx, aoWithHeartbeat)
		iamRolesCreateActivityFuture = workflow.ExecuteActivity(ctx, CreateIamRolesActivityName, activityInput)
	}

	// upload SSH key activity
	var uploadSSHKeyActivityFeature workflow.Future
	if input.UseGeneratedSSHKey {
		{
			activityInput := &UploadSSHKeyActivityInput{
				EKSActivityInput: commonActivityInput,
				SSHKeyName:       GenerateSSHKeyNameForCluster(input.ClusterName),
				SSHSecretID:      input.SSHSecretID,
			}
			uploadSSHKeyActivityFeature = workflow.ExecuteActivity(ctx, UploadSSHKeyActivityName, activityInput)
		}
	}

	// create VPC activity
	var vpcActivityOutput CreateVpcActivityOutput
	{
		activityInput := &CreateVpcActivityInput{
			EKSActivityInput: commonActivityInput,
			VpcID:            input.VpcID,
			RouteTableID:     input.RouteTableID,
			VpcCidr:          input.VpcCidr,
			StackName:        GenerateStackNameForCluster(input.ClusterName),
			Tags:             input.Tags,
		}
		ctx := workflow.WithActivityOptions(ctx, aoWithHeartbeat)
		if err := workflow.ExecuteActivity(ctx, CreateVpcActivityName, activityInput).Get(ctx, &vpcActivityOutput); err != nil {
			return nil, err
		}
	}

	// wait for IAM roles to created before starting user access key creation
	iamRolesActivityOutput := &CreateIamRolesActivityOutput{}
	err := pkgCadence.UnwrapError(iamRolesCreateActivityFuture.Get(ctx, &iamRolesActivityOutput))
	if err != nil {
		return nil, err
	}

	// create IAM user access key, in case defaultUser = false and save as secret
	var userAccessKeyActivityFeature workflow.Future
	{
		activityInput := &CreateClusterUserAccessKeyActivityInput{
			EKSActivityInput: commonActivityInput,
			UserName:         input.ClusterName,
			UseDefaultUser:   input.DefaultUser,
			ClusterUID:       input.ClusterUID,
		}
		userAccessKeyActivityFeature = workflow.ExecuteActivity(ctx, CreateClusterUserAccessKeyActivityName, activityInput)
	}

	// create Subnets activities, gather subnet details for existing subnets activities
	var existingAndNewSubnets []Subnet
	{
		var createSubnetFutures []workflow.Future
		var existingSubnetsIDs []string

		for _, subnet := range input.Subnets {
			if subnet.SubnetID == "" && subnet.Cidr != "" {
				// create new subnet
				activityInput := &CreateSubnetActivityInput{
					EKSActivityInput: commonActivityInput,
					VpcID:            vpcActivityOutput.VpcID,
					RouteTableID:     vpcActivityOutput.RouteTableID,
					SubnetID:         subnet.SubnetID,
					Cidr:             subnet.Cidr,
					AvailabilityZone: subnet.AvailabilityZone,
					StackName:        generateStackNameForSubnet(input.ClusterName, subnet.Cidr),
					Tags:             input.Tags,
				}
				ctx := workflow.WithActivityOptions(ctx, aoWithHeartbeat)
				createSubnetFutures = append(createSubnetFutures, workflow.ExecuteActivity(ctx, CreateSubnetActivityName, activityInput))
			} else if subnet.SubnetID != "" {
				existingSubnetsIDs = append(existingSubnetsIDs, subnet.SubnetID)
			}
		}

		// call get subnet details for existing subnets
		var getSubnetsDetailsFuture workflow.Future
		if len(existingSubnetsIDs) > 0 {
			activityInput := &GetSubnetsDetailsActivityInput{
				OrganizationID: input.OrganizationID,
				SecretID:       input.SecretID,
				Region:         input.Region,
				SubnetIDs:      existingSubnetsIDs,
			}
			getSubnetsDetailsFuture = workflow.ExecuteActivity(ctx, GetSubnetsDetailsActivityName, activityInput)
		}

		// wait for info about newly created subnets
		errs := make([]error, len(createSubnetFutures))
		for i, future := range createSubnetFutures {
			var activityOutput CreateSubnetActivityOutput

			errs[i] = pkgCadence.UnwrapError(future.Get(ctx, &activityOutput))
			if errs[i] == nil {
				existingAndNewSubnets = append(existingAndNewSubnets, Subnet{
					SubnetID:         activityOutput.SubnetID,
					Cidr:             activityOutput.Cidr,
					AvailabilityZone: activityOutput.AvailabilityZone,
				})
			}
		}

		if err := errors.Combine(errs...); err != nil {
			return nil, err
		}

		var getSubnetsDetailsActivityOutput GetSubnetsDetailsActivityOutput
		if getSubnetsDetailsFuture != nil {
			if err := getSubnetsDetailsFuture.Get(ctx, &getSubnetsDetailsActivityOutput); err != nil {
				return nil, err
			}
		}

		for _, subnet := range getSubnetsDetailsActivityOutput.Subnets {
			existingAndNewSubnets = append(existingAndNewSubnets, Subnet{
				SubnetID:         subnet.SubnetID,
				Cidr:             subnet.Cidr,
				AvailabilityZone: subnet.AvailabilityZone,
			})
		}
	}

	userAccessKeyActivityOutput := CreateClusterUserAccessKeyActivityOutput{}
	if err := userAccessKeyActivityFeature.Get(ctx, &userAccessKeyActivityOutput); err != nil {
		return nil, err
	}

	if uploadSSHKeyActivityFeature != nil {
		uploadSSHKeyActivityOutput := &UploadSSHKeyActivityOutput{}
		if err := uploadSSHKeyActivityFeature.Get(ctx, &uploadSSHKeyActivityOutput); err != nil {
			return nil, err
		}
	}

	// create EKS cluster
	{
		activityOutput := CreateEksControlPlaneActivityOutput{}
		activityInput := &CreateEksControlPlaneActivityInput{
			EKSActivityInput:      commonActivityInput,
			KubernetesVersion:     input.KubernetesVersion,
			EncryptionConfig:      input.EncryptionConfig,
			EndpointPrivateAccess: input.EndpointPrivateAccess,
			EndpointPublicAccess:  input.EndpointPublicAccess,
			ClusterRoleArn:        iamRolesActivityOutput.ClusterRoleArn,
			SecurityGroupID:       vpcActivityOutput.SecurityGroupID,
			LogTypes:              input.LogTypes,
			Subnets:               existingAndNewSubnets,
			Tags:                  input.Tags,
		}

		ao := workflow.ActivityOptions{
			ScheduleToStartTimeout: 10 * time.Minute,
			StartToCloseTimeout:    20 * time.Minute,
			WaitForCancellation:    true,
			HeartbeatTimeout:       35 * time.Second,
			RetryPolicy: &cadence.RetryPolicy{
				InitialInterval:          2 * time.Second,
				BackoffCoefficient:       1.5,
				MaximumInterval:          30 * time.Second,
				MaximumAttempts:          5,
				NonRetriableErrorReasons: []string{"cadenceInternal:Panic"},
			},
		}
		ctx := workflow.WithActivityOptions(ctx, ao)
		if err := workflow.ExecuteActivity(ctx, CreateEksControlPlaneActivityName, activityInput).Get(ctx, &activityOutput); err != nil {
			return nil, err
		}
	}

	// initial setup of K8s cluster
	var bootstrapWorkflowFeature workflow.Future
	{
		activityInput := &BootstrapWorkflowInput{
			EKSActivityInput:    commonActivityInput,
			KubernetesVersion:   input.KubernetesVersion,
			NodeInstanceRoleArn: iamRolesActivityOutput.NodeInstanceRoleArn,
			ClusterUserArn:      iamRolesActivityOutput.ClusterUserArn,
			AuthConfigMap:       input.AuthConfigMap,
		}
		bootstrapWorkflowFeature = workflow.ExecuteChildWorkflow(ctx, BootstrapWorkflowName, activityInput)
	}

	{ // Note: create node pools.
		nodePools := make(map[string]eks.NewNodePool, len(input.NodePools))
		nodePoolSubnetErrors := make([]error, 0, len(input.NodePools))
		subnetIDs := make(map[string][]string, len(input.NodePools))
		for _, nodePool := range input.NodePools {
			nodePools[nodePool.Name] = nodePool

			nodePoolSubnets := input.NodePoolSubnets[nodePool.Name]
			nodePoolSubnetIDs := make([]string, 0, len(nodePoolSubnets))
			for _, nodePoolSubnet := range nodePoolSubnets {
				if nodePoolSubnet.SubnetID == "" { // Note: new subnet specified by CIDR.
					for _, clusterSubnet := range existingAndNewSubnets {
						if clusterSubnet.Cidr == nodePoolSubnet.Cidr {
							nodePoolSubnetIDs = append(nodePoolSubnetIDs, clusterSubnet.SubnetID)
						}
					}
				} else { // Note: existing subnet specified by ID.
					nodePoolSubnetIDs = append(nodePoolSubnetIDs, nodePoolSubnet.SubnetID)
				}
			}

			if nodePool.SubnetID != "" {
				nodePoolSubnetIndex := indexStrings(nodePoolSubnetIDs, nodePool.SubnetID)
				if nodePoolSubnetIndex == -1 {
					nodePoolSubnetIDs = append(nodePoolSubnetIDs, nodePool.SubnetID)
				}
			}

			if len(nodePoolSubnetIDs) == 0 {
				nodePoolSubnetErrors = append(
					nodePoolSubnetErrors,
					errors.NewWithDetails(
						"node pool subnet is missing",
						"nodePool", nodePool,
						"nodePoolSubnets", nodePoolSubnets,
						"clusterSubnets", existingAndNewSubnets,
					),
				)
			}

			subnetIDs[nodePool.Name] = nodePoolSubnetIDs
		}
		if len(nodePoolSubnetErrors) != 0 {
			return nil, errors.Combine(nodePoolSubnetErrors...)
		}

		activityInput := CreateNodePoolsWorkflowInput{
			ClusterID:                    input.ClusterID,
			CreatorUserID:                input.CreatorUserID,
			NodePools:                    nodePools,
			NodePoolSubnetIDs:            subnetIDs,
			ShouldCreateNodePoolLabelSet: false, // Note: node pool label set operator is created and synced later in parent EKSCreateClusterWorkflow.
			ShouldStoreNodePool:          false, // Note: stored at LegacyClusterAPI.CreateCluster request parsing.
			ShouldUpdateClusterStatus:    false, // Note: parent EKSCreateClusterWorkflow workflow handles status updates.
		}

		err = workflow.ExecuteChildWorkflow(ctx, CreateNodePoolsWorkflowName, activityInput).Get(ctx, nil)
		if err != nil {
			return nil, err
		}
	}

	// wait for initial cluster setup to terminate
	if err := bootstrapWorkflowFeature.Get(ctx, nil); err != nil {
		return nil, pkgCadence.UnwrapError(err)
	}

	var configSecretID string
	{
		activityInput := SaveK8sConfigActivityInput{
			ClusterID:        input.ClusterID,
			ClusterUID:       input.ClusterUID,
			ClusterName:      input.ClusterName,
			OrganizationID:   input.OrganizationID,
			ProviderSecretID: input.SecretID,
			UserSecretID:     userAccessKeyActivityOutput.SecretID,
			Region:           input.Region,
		}
		future := workflow.ExecuteActivity(ctx, SaveK8sConfigActivityName, activityInput)
		if err := future.Get(ctx, &configSecretID); err != nil {
			return nil, err
		}
	}

	output := CreateInfrastructureWorkflowOutput{
		VpcID:              vpcActivityOutput.VpcID,
		NodeInstanceRoleID: iamRolesActivityOutput.NodeInstanceRoleID,
		Subnets:            existingAndNewSubnets,
		ConfigSecretID:     configSecretID,
	}

	return &output, nil
}
