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
)

const CreateInfraWorkflowName = "eks-create-infra"

// CreateInfrastructureWorkflowInput holds data needed by the create EKS cluster infrastructure workflow
type CreateInfrastructureWorkflowInput struct {
	Region         string
	OrganizationID uint
	SecretID       string
	SSHSecretID    string

	ClusterUID                string
	ClusterName               string
	VpcID                     string
	RouteTableID              string
	VpcCidr                   string
	VpcCloudFormationTemplate string
	ScaleEnabled              bool

	Subnets                      []Subnet
	ASGSubnetMapping             map[string][]Subnet
	SubnetCloudFormationTemplate string

	IAMRolesCloudFormationTemplate string
	DefaultUser                    bool
	ClusterRoleID                  string
	NodeInstanceRoleID             string

	KubernetesVersion     string
	EndpointPrivateAccess bool
	EndpointPublicAccess  bool

	LogTypes                  []string
	ASGCloudFormationTemplate string
	AsgList                   []AutoscaleGroup
}

type CreateInfrastructureWorkflowOutput struct {
	NodeInstanceRoleID string
	Subnets            []Subnet
}

// CreateInfrastructureWorkflow executes the Cadence workflow responsible for creating EKS
// cluster infrastructure such as VPC, subnets, EKS master nodes, worker nodes, etc
func CreateInfrastructureWorkflow(ctx workflow.Context, input CreateInfrastructureWorkflowInput) (*CreateInfrastructureWorkflowOutput, error) {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		WaitForCancellation:    true,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          2 * time.Second,
			BackoffCoefficient:       1.5,
			MaximumInterval:          30 * time.Second,
			MaximumAttempts:          30,
			NonRetriableErrorReasons: []string{"cadenceInternal:Panic"},
		},
	}

	commonActivityInput := EKSActivityInput{
		OrganizationID:            input.OrganizationID,
		SecretID:                  input.SecretID,
		Region:                    input.Region,
		ClusterName:               input.ClusterName,
		AWSClientRequestTokenBase: input.ClusterUID,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	// create IAM roles activity
	var iamRolesCreateActivityFeature workflow.Future
	{
		activityInput := &CreateIamRolesActivityInput{
			EKSActivityInput:       commonActivityInput,
			CloudFormationTemplate: input.IAMRolesCloudFormationTemplate,
			StackName:              generateStackNameForIam(input.ClusterName),
			DefaultUser:            input.DefaultUser,
			ClusterRoleID:          input.ClusterRoleID,
			NodeInstanceRoleID:     input.NodeInstanceRoleID,
		}
		iamRolesCreateActivityFeature = workflow.ExecuteActivity(ctx, CreateIamRolesActivityName, activityInput)
	}

	// create IAM user access key key if not default user and save as secret
	var userAccessKeyActivityFeature workflow.Future
	{
		activityInput := &CreateClusterUserAccessKeyActivityInput{
			EKSActivityInput: commonActivityInput,
			UserName:         input.ClusterName,
			UseDefaultUser:   input.DefaultUser,
		}
		userAccessKeyActivityFeature = workflow.ExecuteActivity(ctx, CreateClusterUserAccessKeyActivityName, activityInput)
	}

	sshKeyName := generateSSHKeyNameForCluster(input.ClusterName)
	// upload SSH key activity
	var uploadSSHKeyActivityFeature workflow.Future
	{
		activityInput := &UploadSSHKeyActivityInput{
			EKSActivityInput: commonActivityInput,
			SSHKeyName:       generateSSHKeyNameForCluster(input.ClusterName),
			SSHSecretID:      input.SSHSecretID,
		}
		uploadSSHKeyActivityFeature = workflow.ExecuteActivity(ctx, UploadSSHKeyActivityName, activityInput)
	}

	// create VPC activity
	var vpcActivityOutput CreateVpcActivityOutput
	{
		activityInput := &CreateVpcActivityInput{
			EKSActivityInput:       commonActivityInput,
			VpcID:                  input.VpcID,
			RouteTableID:           input.RouteTableID,
			VpcCidr:                input.VpcCidr,
			CloudFormationTemplate: input.VpcCloudFormationTemplate,
			StackName:              generateStackNameForCluster(input.ClusterName),
		}
		if err := workflow.ExecuteActivity(ctx, CreateVpcActivityName, activityInput).Get(ctx, &vpcActivityOutput); err != nil {
			return nil, err
		}
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
					EKSActivityInput:       commonActivityInput,
					VpcID:                  vpcActivityOutput.VpcID,
					RouteTableID:           vpcActivityOutput.RouteTableID,
					SubnetID:               subnet.SubnetID,
					Cidr:                   subnet.Cidr,
					AvailabilityZone:       subnet.AvailabilityZone,
					CloudFormationTemplate: input.SubnetCloudFormationTemplate,
					StackName:              generateStackNameForSubnet(input.ClusterName, subnet.Cidr),
				}

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

		// wait for subnet info
		errs := make([]error, len(createSubnetFutures))
		for i, future := range createSubnetFutures {
			var activityOutput CreateSubnetActivityOutput

			errs[i] = future.Get(ctx, &activityOutput)
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

	iamRolesActivityOutput := &CreateIamRolesActivityOutput{}
	if err := iamRolesCreateActivityFeature.Get(ctx, &iamRolesActivityOutput); err != nil {
		return nil, err
	}

	userAccessKeyActivityOutput := CreateClusterUserAccessKeyActivityOutput{}
	if err := userAccessKeyActivityFeature.Get(ctx, &userAccessKeyActivityOutput); err != nil {
		return nil, err
	}

	uploadSSHKeyActivityOutput := &UploadSSHKeyActivityOutput{}
	if err := uploadSSHKeyActivityFeature.Get(ctx, &uploadSSHKeyActivityOutput); err != nil {
		return nil, err
	}

	// create EKS cluster
	{
		activityOutput := CreateEksControlPlaneActivityOutput{}
		activityInput := &CreateEksControlPlaneActivityInput{
			EKSActivityInput:      commonActivityInput,
			KubernetesVersion:     input.KubernetesVersion,
			EndpointPrivateAccess: input.EndpointPrivateAccess,
			EndpointPublicAccess:  input.EndpointPublicAccess,
			ClusterRoleArn:        iamRolesActivityOutput.ClusterRoleArn,
			SecurityGroupID:       vpcActivityOutput.SecurityGroupID,
			LogTypes:              input.LogTypes,
			Subnets:               existingAndNewSubnets,
		}

		ao := workflow.ActivityOptions{
			ScheduleToStartTimeout: 10 * time.Minute,
			StartToCloseTimeout:    20 * time.Minute,
			WaitForCancellation:    true,
			RetryPolicy: &cadence.RetryPolicy{
				InitialInterval:          2 * time.Second,
				BackoffCoefficient:       1.5,
				MaximumInterval:          30 * time.Second,
				MaximumAttempts:          30,
				NonRetriableErrorReasons: []string{"cadenceInternal:Panic"},
			},
		}
		ctx := workflow.WithActivityOptions(ctx, ao)
		if err := workflow.ExecuteActivity(ctx, CreateEksControlPlaneActivityName, activityInput).Get(ctx, &activityOutput); err != nil {
			return nil, err
		}
	}

	// initial setup of K8s cluster
	var bootstrapActivityFeature workflow.Future
	{
		activityInput := &BootstrapActivityInput{
			EKSActivityInput:    commonActivityInput,
			KubernetesVersion:   input.KubernetesVersion,
			NodeInstanceRoleArn: iamRolesActivityOutput.NodeInstanceRoleArn,
			ClusterUserArn:      iamRolesActivityOutput.ClusterUserArn,
		}
		bootstrapActivityFeature = workflow.ExecuteActivity(ctx, BootstrapActivityName, activityInput)
	}

	// create AutoScalingGroups
	asgFutures := make([]workflow.Future, 0)
	for _, asg := range input.AsgList {

		asgSubnets := input.ASGSubnetMapping[asg.Name]
		for i := range asgSubnets {
			for _, sn := range existingAndNewSubnets {
				if (asgSubnets[i].SubnetID == "" && sn.Cidr == asgSubnets[i].Cidr) ||
					(asgSubnets[i].SubnetID != "" && sn.SubnetID == asgSubnets[i].SubnetID) {
					asgSubnets[i].SubnetID = sn.SubnetID
					asgSubnets[i].Cidr = sn.Cidr
					asgSubnets[i].AvailabilityZone = sn.AvailabilityZone
				}
			}
		}

		activityInput := CreateAsgActivityInput{
			EKSActivityInput:       commonActivityInput,
			CloudFormationTemplate: input.ASGCloudFormationTemplate,
			StackName:              generateNodePoolStackName(input.ClusterName, asg.Name),

			ScaleEnabled: input.ScaleEnabled,
			SSHKeyName:   sshKeyName,

			Subnets: asgSubnets,

			VpcID:               vpcActivityOutput.VpcID,
			SecurityGroupID:     vpcActivityOutput.SecurityGroupID,
			NodeSecurityGroupID: vpcActivityOutput.NodeSecurityGroupID,
			NodeInstanceRoleID:  iamRolesActivityOutput.NodeInstanceRoleID,

			Name:             asg.Name,
			NodeSpotPrice:    asg.NodeSpotPrice,
			Autoscaling:      asg.Autoscaling,
			NodeMinCount:     asg.NodeMinCount,
			NodeMaxCount:     asg.NodeMaxCount,
			Count:            asg.Count,
			NodeImage:        asg.NodeImage,
			NodeInstanceType: asg.NodeInstanceType,
			Labels:           asg.Labels,
		}
		f := workflow.ExecuteActivity(ctx, CreateAsgActivityName, activityInput)
		asgFutures = append(asgFutures, f)
	}

	// wait for AutoScalingGroups to be created
	errs := make([]error, len(asgFutures))
	for i, future := range asgFutures {
		var activityOutput CreateAsgActivityOutput
		errs[i] = future.Get(ctx, &activityOutput)
	}
	if err := errors.Combine(errs...); err != nil {
		return nil, err
	}

	// wait for initial cluster setup to terminate
	bootstrapActivityOutput := &BootstrapActivityOutput{}
	if err := bootstrapActivityFeature.Get(ctx, &bootstrapActivityOutput); err != nil {
		return nil, err
	}

	output := CreateInfrastructureWorkflowOutput{
		NodeInstanceRoleID: iamRolesActivityOutput.NodeInstanceRoleID,
		Subnets:            existingAndNewSubnets,
	}

	return &output, nil
}
