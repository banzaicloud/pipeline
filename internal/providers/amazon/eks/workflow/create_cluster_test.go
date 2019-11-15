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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/testsuite"
	"go.uber.org/cadence/workflow"
)

type CreateClusterWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment
}

func TestCreateClusterWorkflowTestSuite(t *testing.T) {
	workflow.RegisterWithOptions(CreateClusterWorkflow, workflow.RegisterOptions{Name: CreateClusterWorkflowName})
	workflow.RegisterWithOptions(CreateInfrastructureWorkflow, workflow.RegisterOptions{Name: CreateInfraWorkflowName})

	createVPCActivity := NewCreateVPCActivity(nil, "")
	activity.RegisterWithOptions(createVPCActivity.Execute, activity.RegisterOptions{Name: CreateVpcActivityName})

	createSubnetActivity := NewCreateSubnetActivity(nil, "")
	activity.RegisterWithOptions(createSubnetActivity.Execute, activity.RegisterOptions{Name: CreateSubnetActivityName})

	getSubnetsDetailsActivity := NewGetSubnetsDetailsActivity(nil)
	activity.RegisterWithOptions(getSubnetsDetailsActivity.Execute, activity.RegisterOptions{Name: GetSubnetsDetailsActivityName})

	createIamRolesActivity := NewCreateIamRolesActivity(nil, "")
	activity.RegisterWithOptions(createIamRolesActivity.Execute, activity.RegisterOptions{Name: CreateIamRolesActivityName})

	uploadSSHActivityActivity := NewUploadSSHKeyActivity(nil)
	activity.RegisterWithOptions(uploadSSHActivityActivity.Execute, activity.RegisterOptions{Name: UploadSSHKeyActivityName})

	createEksClusterActivity := NewCreateEksClusterActivity(nil)
	activity.RegisterWithOptions(createEksClusterActivity.Execute, activity.RegisterOptions{Name: CreateEksControlPlaneActivityName})

	waitAttempts := 1
	waitInterval := 20 * time.Second

	createAsgActivity := NewCreateAsgActivity(nil, "", waitAttempts, waitInterval)
	activity.RegisterWithOptions(createAsgActivity.Execute, activity.RegisterOptions{Name: CreateAsgActivityName})

	createUserAccessKeyActivity := NewCreateClusterUserAccessKeyActivity(nil)
	activity.RegisterWithOptions(createUserAccessKeyActivity.Execute, activity.RegisterOptions{Name: CreateClusterUserAccessKeyActivityName})

	bootstrapActivity := NewBootstrapActivity(nil)
	activity.RegisterWithOptions(bootstrapActivity.Execute, activity.RegisterOptions{Name: BootstrapActivityName})

	suite.Run(t, new(CreateClusterWorkflowTestSuite))
}

func (s *CreateClusterWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
}

func (s *CreateClusterWorkflowTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

func (s *CreateClusterWorkflowTestSuite) Test_Successful_Create() {

	workflowInput := CreateClusterWorkflowInput{
		CreateInfrastructureWorkflowInput: CreateInfrastructureWorkflowInput{
			Region:                "us-west-1",
			OrganizationID:        1,
			SecretID:              "my-secret-id",
			SSHSecretID:           "ssh-secret-id",
			ClusterUID:            "cluster-id",
			ClusterName:           "test-cluster-name",
			VpcID:                 "",
			RouteTableID:          "",
			VpcCidr:               "",
			ScaleEnabled:          false,
			DefaultUser:           false,
			ClusterRoleID:         "test-cluster-role-id",
			NodeInstanceRoleID:    "test-node-instance-role-id",
			KubernetesVersion:     "1.14",
			LogTypes:              []string{"test-log-type"},
			EndpointPublicAccess:  true,
			EndpointPrivateAccess: false,
			Subnets: []Subnet{
				{Cidr: "cidr1", AvailabilityZone: "az1"},
				{Cidr: "cidr2", AvailabilityZone: "az2"},
				{SubnetID: "subnet3"},
			},
			ASGSubnetMapping: map[string][]Subnet{
				"pool1": {
					{Cidr: "cidr1", AvailabilityZone: "az1"},
					{Cidr: "cidr2", AvailabilityZone: "az2"},
				},
				"pool2": {{SubnetID: "subnet3"}},
			},
			AsgList: []AutoscaleGroup{
				{
					Name:             "pool1",
					NodeSpotPrice:    "0.2",
					Autoscaling:      true,
					NodeMinCount:     2,
					NodeMaxCount:     3,
					Count:            2,
					NodeImage:        "ami-test1",
					NodeInstanceType: "vm-type1-test",
					Labels: map[string]string{
						"test-label1":         "test-value1",
						"test-label2.io/name": "test-value2",
					},
				},
				{
					Name:             "pool2",
					NodeSpotPrice:    "0.0",
					Autoscaling:      false,
					NodeMinCount:     3,
					NodeMaxCount:     3,
					Count:            3,
					NodeImage:        "ami-test2",
					NodeInstanceType: "vm-type2-test",
				},
			},
		},
		ClusterID: 1,
	}

	eksActivity := EKSActivityInput{
		OrganizationID:            workflowInput.OrganizationID,
		SecretID:                  workflowInput.SecretID,
		Region:                    workflowInput.Region,
		ClusterName:               workflowInput.ClusterName,
		AWSClientRequestTokenBase: workflowInput.ClusterUID,
	}

	s.env.OnActivity(CreateIamRolesActivityName, mock.Anything, CreateIamRolesActivityInput{
		EKSActivityInput:   eksActivity,
		StackName:          "pipeline-eks-iam-test-cluster-name",
		DefaultUser:        workflowInput.DefaultUser,
		ClusterRoleID:      workflowInput.ClusterRoleID,
		NodeInstanceRoleID: workflowInput.NodeInstanceRoleID,
	},
	).Return(&CreateIamRolesActivityOutput{
		ClusterRoleArn:      "cluster-role-arn",
		ClusterUserArn:      "cluster-user-arn",
		NodeInstanceRoleID:  "node-instance-role-id",
		NodeInstanceRoleArn: "node-instance-role-arn",
	}, nil)

	s.env.OnActivity(CreateClusterUserAccessKeyActivityName, mock.Anything, CreateClusterUserAccessKeyActivityInput{
		EKSActivityInput: eksActivity,
		UserName:         "test-cluster-name",
		UseDefaultUser:   false,
	}).Return(&CreateClusterUserAccessKeyActivityOutput{}, nil)

	s.env.OnActivity(UploadSSHKeyActivityName, mock.Anything, UploadSSHKeyActivityInput{
		EKSActivityInput: eksActivity,
		SSHKeyName:       "pipeline-eks-ssh-test-cluster-name",
		SSHSecretID:      "ssh-secret-id",
	}).Return(&UploadSSHKeyActivityOutput{}, nil)

	s.env.OnActivity(CreateVpcActivityName, mock.Anything, CreateVpcActivityInput{
		EKSActivityInput: eksActivity,
		StackName:        "pipeline-eks-test-cluster-name",
	}).Return(&CreateVpcActivityOutput{
		VpcID:               "new-vpc-id",
		RouteTableID:        "new-route-table-id",
		SecurityGroupID:     "test-eks-controlplane-security-group-id",
		NodeSecurityGroupID: "test-node-securitygroup-id",
	}, nil)

	s.env.OnActivity(CreateSubnetActivityName, mock.Anything, CreateSubnetActivityInput{
		EKSActivityInput: eksActivity,
		Cidr:             "cidr1",
		AvailabilityZone: "az1",
		StackName:        "pipeline-eks-subnet-test-cluster-name-cidr1",
		VpcID:            "new-vpc-id",
		RouteTableID:     "new-route-table-id",
	}).Return(&CreateSubnetActivityOutput{
		SubnetID:         "subnet1",
		Cidr:             "cidr1",
		AvailabilityZone: "az1",
	}, nil).Once()

	s.env.OnActivity(CreateSubnetActivityName, mock.Anything, CreateSubnetActivityInput{
		EKSActivityInput: eksActivity,
		Cidr:             "cidr2",
		AvailabilityZone: "az2",
		StackName:        "pipeline-eks-subnet-test-cluster-name-cidr2",
		VpcID:            "new-vpc-id",
		RouteTableID:     "new-route-table-id",
	}).Return(&CreateSubnetActivityOutput{
		SubnetID:         "subnet2",
		Cidr:             "cidr2",
		AvailabilityZone: "az2",
	}, nil).Once()

	s.env.OnActivity(GetSubnetsDetailsActivityName, mock.Anything, GetSubnetsDetailsActivityInput{
		OrganizationID: 1,
		SecretID:       "my-secret-id",
		Region:         "us-west-1",
		SubnetIDs: []string{
			"subnet3",
		},
	}).Return(&GetSubnetsDetailsActivityOutput{
		Subnets: []Subnet{
			{
				SubnetID:         "subnet3",
				Cidr:             "cidr3",
				AvailabilityZone: "az3",
			},
		},
	}, nil).Once()

	s.env.OnActivity(CreateEksControlPlaneActivityName, mock.Anything, CreateEksControlPlaneActivityInput{
		EKSActivityInput:      eksActivity,
		KubernetesVersion:     "1.14",
		EndpointPrivateAccess: true,
		EndpointPublicAccess:  true,
		ClusterRoleArn:        "cluster-role-arn",
		SecurityGroupID:       "test-eks-controlplane-security-group-id",
		LogTypes: []string{
			"test-log-type",
		},
		Subnets: []Subnet{
			{
				SubnetID:         "subnet1",
				Cidr:             "cidr1",
				AvailabilityZone: "az1",
			},
			{
				SubnetID:         "subnet2",
				Cidr:             "cidr2",
				AvailabilityZone: "az2",
			},
			{
				SubnetID:         "subnet3",
				Cidr:             "cidr3",
				AvailabilityZone: "az3",
			},
		},
	}).Return(&CreateEksControlPlaneActivityOutput{}, nil)

	s.env.OnActivity(CreateAsgActivityName, mock.Anything, CreateAsgActivityInput{
		EKSActivityInput:    eksActivity,
		StackName:           "pipeline-eks-nodepool-test-cluster-name-pool1",
		VpcID:               "new-vpc-id",
		SecurityGroupID:     "test-eks-controlplane-security-group-id",
		NodeSecurityGroupID: "test-node-securitygroup-id",
		NodeInstanceRoleID:  "node-instance-role-id",
		SSHKeyName:          "pipeline-eks-ssh-test-cluster-name",
		Name:                "pool1",
		NodeSpotPrice:       "0.2",
		Autoscaling:         true,
		NodeMinCount:        2,
		NodeMaxCount:        3,
		Count:               2,
		NodeImage:           "ami-test1",
		NodeInstanceType:    "vm-type1-test",
		Labels: map[string]string{
			"test-label1":         "test-value1",
			"test-label2.io/name": "test-value2",
		},
		Subnets: []Subnet{
			{
				SubnetID:         "subnet1",
				Cidr:             "cidr1",
				AvailabilityZone: "az1",
			},
			{
				SubnetID:         "subnet2",
				Cidr:             "cidr2",
				AvailabilityZone: "az2",
			},
		},
	}).Return(&CreateAsgActivityOutput{}, nil).Once()

	s.env.OnActivity(CreateAsgActivityName, mock.Anything, CreateAsgActivityInput{
		EKSActivityInput:    eksActivity,
		StackName:           "pipeline-eks-nodepool-test-cluster-name-pool2",
		VpcID:               "new-vpc-id",
		SecurityGroupID:     "test-eks-controlplane-security-group-id",
		NodeSecurityGroupID: "test-node-securitygroup-id",
		NodeInstanceRoleID:  "node-instance-role-id",
		SSHKeyName:          "pipeline-eks-ssh-test-cluster-name",
		Name:                "pool2",
		NodeSpotPrice:       "0.0",
		Autoscaling:         false,
		NodeMinCount:        3,
		NodeMaxCount:        3,
		Count:               3,
		NodeImage:           "ami-test2",
		NodeInstanceType:    "vm-type2-test",
		Subnets: []Subnet{
			{
				SubnetID:         "subnet3",
				Cidr:             "cidr3",
				AvailabilityZone: "az3",
			},
		},
	}).Return(&CreateAsgActivityOutput{}, nil).Once()

	s.env.OnActivity(BootstrapActivityName, mock.Anything, BootstrapActivityInput{
		EKSActivityInput:    eksActivity,
		KubernetesVersion:   "1.14",
		NodeInstanceRoleArn: "node-instance-role-arn",
		ClusterUserArn:      "cluster-user-arn",
	}).Return(&BootstrapActivityOutput{}, nil)

	s.env.ExecuteWorkflow(CreateClusterWorkflowName, workflowInput)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var workflowOutput CreateClusterWorkflowOutput
	s.env.GetWorkflowResult(&workflowOutput) //nolint: errcheck

	expectedWorkflowOutput := CreateClusterWorkflowOutput{
		CreateInfrastructureWorkflowOutput{
			VpcID:              "new-vpc-id",
			NodeInstanceRoleID: "node-instance-role-id",
			Subnets: []Subnet{
				{SubnetID: "subnet1", Cidr: "cidr1", AvailabilityZone: "az1"},
				{SubnetID: "subnet2", Cidr: "cidr2", AvailabilityZone: "az2"},
				{SubnetID: "subnet3", Cidr: "cidr3", AvailabilityZone: "az3"},
			},
		},
	}
	s.Equal(expectedWorkflowOutput, workflowOutput)
}

func (s *CreateClusterWorkflowTestSuite) Test_Successful_Fail_To_Create_VPC() {

	workflowInput := CreateClusterWorkflowInput{
		CreateInfrastructureWorkflowInput: CreateInfrastructureWorkflowInput{
			Region:                "us-west-1",
			OrganizationID:        1,
			SecretID:              "my-secret-id",
			SSHSecretID:           "ssh-secret-id",
			ClusterUID:            "cluster-id",
			ClusterName:           "test-cluster-name",
			VpcID:                 "",
			RouteTableID:          "",
			VpcCidr:               "",
			ScaleEnabled:          false,
			DefaultUser:           false,
			ClusterRoleID:         "test-cluster-role-id",
			NodeInstanceRoleID:    "test-node-instance-role-id",
			KubernetesVersion:     "1.14",
			LogTypes:              []string{"test-log-type"},
			EndpointPublicAccess:  true,
			EndpointPrivateAccess: false,
			Subnets: []Subnet{
				{Cidr: "cidr1", AvailabilityZone: "az1"},
				{Cidr: "cidr2", AvailabilityZone: "az2"},
				{SubnetID: "subnet3"},
			},
			ASGSubnetMapping: map[string][]Subnet{
				"pool1": {
					{Cidr: "cidr1", AvailabilityZone: "az1"},
					{Cidr: "cidr2", AvailabilityZone: "az2"},
				},
				"pool2": {{SubnetID: "subnet3"}},
			},
			AsgList: []AutoscaleGroup{
				{
					Name:             "pool1",
					NodeSpotPrice:    "0.2",
					Autoscaling:      true,
					NodeMinCount:     2,
					NodeMaxCount:     3,
					Count:            2,
					NodeImage:        "ami-test1",
					NodeInstanceType: "vm-type1-test",
					Labels: map[string]string{
						"test-label1":         "test-value1",
						"test-label2.io/name": "test-value2",
					},
				},
				{
					Name:             "pool2",
					NodeSpotPrice:    "0.0",
					Autoscaling:      false,
					NodeMinCount:     3,
					NodeMaxCount:     3,
					Count:            3,
					NodeImage:        "ami-test2",
					NodeInstanceType: "vm-type2-test",
				},
			},
		},
		ClusterID: 1,
	}

	s.env.OnActivity(CreateIamRolesActivityName, mock.Anything, mock.Anything).Return(&CreateIamRolesActivityOutput{
		ClusterRoleArn:      "cluster-role-arn",
		ClusterUserArn:      "cluster-user-arn",
		NodeInstanceRoleID:  "node-instance-role-id",
		NodeInstanceRoleArn: "node-instance-role-arn",
	}, nil)

	s.env.OnActivity(CreateClusterUserAccessKeyActivityName, mock.Anything, mock.Anything).Return(&CreateClusterUserAccessKeyActivityOutput{}, nil)

	s.env.OnActivity(UploadSSHKeyActivityName, mock.Anything, mock.Anything).Return(&UploadSSHKeyActivityOutput{}, nil)

	s.env.OnActivity(CreateVpcActivityName, mock.Anything, mock.Anything).Return(nil, errors.New("failed to create VPC"))

	s.env.ExecuteWorkflow(CreateClusterWorkflowName, workflowInput)

	s.True(s.env.IsWorkflowCompleted())
	s.Error(s.env.GetWorkflowError())
}
