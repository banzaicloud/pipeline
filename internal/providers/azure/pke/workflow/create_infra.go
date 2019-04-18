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

	"go.uber.org/cadence/workflow"
)

const CreateInfraWorkflowName = "pke-azure-create-infra"

type CreateAzureInfrastructureWorkflowInput struct {
	OrganizationID    uint
	ClusterName       string
	SecretID          string
	Location          string
	ResourceGroupName string
}

func CreateInfrastructureWorkflow(ctx workflow.Context, input CreateAzureInfrastructureWorkflowInput) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		ScheduleToCloseTimeout: 15 * time.Minute,
		WaitForCancellation:    true,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	// Create master network security group
	var masterNSGOutput CreateNSGActivityOutput
	{
		activityInput := CreateNSGActivityInput{
			OrganizationID:    input.OrganizationID,
			SecretID:          input.SecretID,
			ClusterName:       input.ClusterName,
			ResourceGroupName: input.ResourceGroupName,
			SecurityGroup: SecurityGroup{
				Name:     input.ClusterName + "-nsg-master",
				Location: input.Location,
				Rules: []SecurityRule{
					{
						Access:               "Allow",
						Description:          "Allow K8s API inbound",
						Destination:          "*",
						DestinationPortRange: "6443",
						Direction:            "Inbound",
						Priority:             1001,
						Protocol:             "Tcp",
						Source:               "*",
						SourcePortRange:      "*",
					},
				},
			},
		}
		err := workflow.ExecuteActivity(ctx, CreateNSGActivityName, activityInput).Get(ctx, &masterNSGOutput)
		if err != nil {
			return err
		}
	}

	// Create worker network security group
	var workerNSGOutput CreateNSGActivityOutput
	{
		activityInput := CreateNSGActivityInput{
			OrganizationID:    input.OrganizationID,
			SecretID:          input.SecretID,
			ClusterName:       input.ClusterName,
			ResourceGroupName: input.ResourceGroupName,
			SecurityGroup: SecurityGroup{
				Name:     input.ClusterName + "-nsg-worker",
				Location: input.Location,
				Rules:    []SecurityRule{},
			},
		}
		err := workflow.ExecuteActivity(ctx, CreateNSGActivityName, activityInput).Get(ctx, &masterNSGOutput)
		if err != nil {
			return err
		}
	}

	// Create route table
	var createRouteTableOutput CreateRouteTableActivityOutput
	{
		activityInput := CreateRouteTableActivityInput{
			OrganizationID:    input.OrganizationID,
			SecretID:          input.SecretID,
			ClusterName:       input.ClusterName,
			ResourceGroupName: input.ResourceGroupName,
			RouteTable: RouteTable{
				Name:     input.ClusterName + "-route-table",
				Location: input.Location,
			},
		}
		err := workflow.ExecuteActivity(ctx, CreateRouteTableActivityName, activityInput).Get(ctx, &createRouteTableOutput)
		if err != nil {
			return err
		}
	}

	// Create virtual network and subnets
	// TODO review these CIDR etc values
	var createVnetOutput CreateVnetActivityOutput
	{
		activityInput := CreateVnetActivityInput{
			ResourceGroupName: input.ResourceGroupName,
			OrganizationID:    input.OrganizationID,
			ClusterName:       input.ClusterName,
			SecretID:          input.SecretID,
			VirtualNetwork: VirtualNetwork{
				Name:     input.ClusterName + "-vnet",
				CIDRs:    []string{"10.240.0.0/16"},
				Location: input.Location,
				Subnets: []Subnet{
					{
						Name:                   input.ClusterName + "-subnet-master",
						CIDR:                   "10.240.0.0/24",
						NetworkSecurityGroupID: masterNSGOutput.NetworkSecurityGroupID,
						RouteTableID:           createRouteTableOutput.RouteTableID,
					},
					{
						Name:                   input.ClusterName + "-subnet-worker",
						CIDR:                   "10.240.1.0/24",
						NetworkSecurityGroupID: workerNSGOutput.NetworkSecurityGroupID,
						RouteTableID:           createRouteTableOutput.RouteTableID,
					},
				},
			},
		}
		err := workflow.ExecuteActivity(ctx, CreateVnetActivityName, activityInput).Get(ctx, &createVnetOutput)
		if err != nil {
			return err
		}
	}

	// Create basic load balancer
	var createLBOutput CreateLoadBalancerActivityOutput
	{
		bap := BackendAddressPool{
			Name: "backend-pool-master",
		}
		fic := FrontendIPConfiguration{
			Name: "frontend-ip-config",
			PublicIPAddress: PublicIPAddress{
				Location: input.Location,
				Name:     input.ClusterName + "-pip-in",
				SKU:      "Standard",
			},
		}
		probe := Probe{
			Name:     "api-server-probe",
			Port:     int32(6443),
			Protocol: "Tcp",
		}
		activityInput := CreateLoadBalancerActivityInput{
			LoadBalancer: LoadBalancer{
				Name:     "kubernetes",
				Location: input.Location,
				SKU:      "Standard",
				BackendAddressPools: []BackendAddressPool{
					bap,
				},
				FrontendIPConfigurations: []FrontendIPConfiguration{
					fic,
				},
				InboundNATPools: []InboundNATPool{
					{
						Name:                   "ssh-in-nat-pool",
						BackendPort:            int32(22),
						FrontendIPConfig:       &fic,
						FrontendPortRangeEnd:   int32(50000),
						FrontendPortRangeStart: int32(50010),
						Protocol:               "Tcp",
					},
				},
				LoadBalancingRules: []LoadBalancingRule{
					{
						Name:                "api-server-rule",
						BackendAddressPool:  &bap,
						BackendPort:         int32(6443),
						DisableOutboundSNAT: false,
						FrontendIPConfig:    &fic,
						FrontendPort:        int32(6443),
						Probe:               &probe,
						Protocol:            "Tcp",
					},
				},
				Probes: []Probe{
					probe,
				},
			},
			ResourceGroupName: input.ResourceGroupName,
			OrganizationID:    input.OrganizationID,
			ClusterName:       input.ClusterName,
			SecretID:          input.SecretID,
		}

		err := workflow.ExecuteActivity(ctx, CreateLoadBalancerActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	// Create master scale set
	{
		activityInput := CreateVMSSActivityInput{
			OrganizationID:    input.OrganizationID,
			SecretID:          input.SecretID,
			ClusterName:       input.ClusterName,
			ResourceGroupName: input.ResourceGroupName,
			ScaleSet: VirtualMachineScaleSet{
				AdminUsername:          "pipeline",
				InstanceCount:          int64(1),
				InstanceType:           "Standard_B2s",
				LBBackendAddressPoolID: createLBOutput.BackendAddressPoolIDs["backend-pool-master"],
				LBInboundNATPoolID:     createLBOutput.InboundNATPoolIDs["ssh-in-nat-pool"],
				Location:               input.Location,
				Name:                   input.ClusterName + "-controller-vmss",
				NetworkSecurityGroupID: masterNSGOutput.NetworkSecurityGroupID,
				SSHPublicKey:           "???", // TODO
				SubnetID:               createVnetOutput.SubnetIDs[input.ClusterName+"-subnet-1"],
				UserDataScript:         "???", // TODO
				Zones:                  []string{"1"},
			},
		}

		err := workflow.ExecuteActivity(ctx, CreateVMSSActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	// Set AssignRolePolicy
	return nil
}
