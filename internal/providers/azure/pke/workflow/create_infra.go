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

	"github.com/gofrs/uuid"
	"go.uber.org/cadence/workflow"
)

const CreateInfraWorkflowName = "pke-azure-create-infra"

type CreateAzureInfrastructureWorkflowInput struct {
	PipelineURL                  string
	OrganizationID               uint
	ClusterID                    uint
	ClusterName                  string
	SecretID                     string
	Location                     string
	ResourceGroupName            string
	TenantID                     string
	SSHPublicKey                 string
	MasterUserDataScriptTemplate string
	WorkerUserDataScriptTemplate string
}

// MasterUserDataScriptTemplate is the template to execute PKE command to initialise cluster

const MasterUserDataScriptTemplate = `#!/bin/sh
# TODO: make IP obtainment more robust
export PRIVATE_IP=$(hostname -I | cut -d" " -f 1)
curl -v https://banzaicloud.com/downloads/pke/pke-{{ .PKEVersion}} -o /usr/local/bin/pke
chmod +x /usr/local/bin/pke
export PATH=$PATH:/usr/local/bin/
	
pke install master --pipeline-url="{{ .PipelineURL }}" \
--pipeline-token="{{ .PipelineToken }}" \
--pipeline-org-id={{ .OrgID }} \
--pipeline-cluster-id={{ .ClusterID}} \
--pipeline-nodepool={{ .NodePoolName }} \
--kubernetes-cloud-provider=azure \
--azure-tenant-id={{ .TenantID }} \
--azure-subnet-name={{ .SubnetName }} \
--azure-security-group-name={{ .NSGName }} \
--azure-vnet-name={{ .VnetName }} \
--azure-vnet-resource-group={{ .VnetResourceGroupName }} \
--azure-vm-type=vmss \
--azure-loadbalancer-sku=standard \
--azure-route-table-name={{ .RouteTableName }} \
--kubernetes-advertise-address=$PRIVATE_IP:6443 \
--kubernetes-api-server=$PRIVATE_IP:6443 \
--kubernetes-infrastructure-cidr={{ .InfraCIDR }} \
--kubernetes-api-server-cert-sans={{ .PublicAddress }}`

const WorkerUserDataScriptTemplate = `
#!/bin/sh
# TODO: make IP obtainment more robust
export PRIVATE_IP=$(hostname -I | cut -d" " -f 1)
curl -v https://banzaicloud.com/downloads/pke/pke-{{ .PKEVersion }} -o /usr/local/bin/pke
chmod +x /usr/local/bin/pke
export PATH=$PATH:/usr/local/bin/
	
pke install worker --pipeline-url="{{ .PipelineURL }}" \
--pipeline-token="{{ .PipelineToken }}" \
--pipeline-org-id={{ .OrgID }} \
--pipeline-cluster-id={{ .ClusterID}} \
--pipeline-nodepool={{ .NodePoolName }} \
--kubernetes-cloud-provider=azure \
--azure-tenant-id={{ .TenantID }} \
--azure-subnet-name={{ .SubnetName }} \
--azure-security-group-name={{ .NSGName }} \
--azure-vnet-name={{ .VnetName }} \
--azure-vnet-resource-group={{ .VnetResourceGroupName }} \
--azure-vm-type=standard \
--azure-loadbalancer-sku=standard \
--azure-route-table-name={{ .RouteTableName }} \
--kubernetes-api-server=$PRIVATEIP:6443 \
--kubernetes-infrastructure-cidr={{ .InfraCIDR }} \
--kubernetes-pod-network-cidr=""
`

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
						Name:                 "server-allow-ssh-inbound",
						Access:               "Allow",
						Description:          "Allow SSH server inbound connections",
						Destination:          "*",
						DestinationPortRange: "22",
						Direction:            "Inbound",
						Priority:             1000,
						Protocol:             "Tcp",
						Source:               "*",
						SourcePortRange:      "*",
					},
					{
						Name:                 "kubernetes-allow-api-server-inbound",
						Access:               "Allow",
						Description:          "Allow K8s API server inbound connections",
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
		err := workflow.ExecuteActivity(ctx, CreateNSGActivityName, activityInput).Get(ctx, &workerNSGOutput)
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
	var createPublicIPOutput CreatePublicIPActivityOutput
	// Create PublicIP
	{
		activityInput := CreatePublicIPActivityInput{
			OrganizationID:    input.OrganizationID,
			SecretID:          input.SecretID,
			ClusterName:       input.ClusterName,
			ResourceGroupName: input.ResourceGroupName,
			PublicIPAddress: PublicIPAddress{
				Location: input.Location,
				Name:     input.ClusterName + "-pip-in",
				SKU:      "Standard",
			},
		}
		err := workflow.ExecuteActivity(ctx, CreatePublicIPActivityName, activityInput).Get(ctx, &createPublicIPOutput)
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
			Name:              "frontend-ip-config",
			PublicIPAddressID: createPublicIPOutput.PublicIPAddressID,
		}
		probe := Probe{
			Name:     "api-server-probe",
			Port:     6443,
			Protocol: "Tcp",
		}
		activityInput := CreateLoadBalancerActivityInput{
			LoadBalancer: LoadBalancer{
				Name:     "kubernetes", // TODO: lb name should be unique per cluster unless it's shared by multiple clusters
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
						Name:                   "ssh-inbound-nat-pool",
						BackendPort:            22,
						FrontendIPConfig:       &fic,
						FrontendPortRangeEnd:   50100,
						FrontendPortRangeStart: 50000,
						Protocol:               "Tcp",
					},
				},
				LoadBalancingRules: []LoadBalancingRule{
					{
						Name:                "api-server-rule",
						BackendAddressPool:  &bap,
						BackendPort:         6443,
						DisableOutboundSNAT: false,
						FrontendIPConfig:    &fic,
						FrontendPort:        6443,
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

		err := workflow.ExecuteActivity(ctx, CreateLoadBalancerActivityName, activityInput).Get(ctx, &createLBOutput)
		if err != nil {
			return err
		}
	}

	// Create master scale set
	var masterVMSSOutput CreateVMSSActivityOutput
	{
		activityInput := CreateVMSSActivityInput{
			OrganizationID:         input.OrganizationID,
			ClusterID:              input.ClusterID,
			SecretID:               input.SecretID,
			ClusterName:            input.ClusterName,
			ResourceGroupName:      input.ResourceGroupName,
			UserDataScriptTemplate: MasterUserDataScriptTemplate,
			UserDataScriptParams: map[string]string{
				"PipelineURL":           input.PipelineURL,
				"TenantID":              input.TenantID,
				"SubnetName":            input.ClusterName + "-subnet-worker",
				"NSGName":               input.ClusterName + "-nsg-worker",
				"VnetName":              input.ClusterName + "-vnet",
				"VnetResourceGroupName": input.ResourceGroupName,
				"RouteTableName":        input.ClusterName + "-route-table",
				"InfraCIDR":             "10.240.0.0/16",
				"PublicAddress":         createPublicIPOutput.PublicIPAddress,
			},
			ScaleSet: VirtualMachineScaleSet{
				AdminUsername: "azureuser",
				Image: Image{
					Offer:     "CentOS-CI",
					Publisher: "OpenLogic",
					SKU:       "7-CI",
					Version:   "7.6.20190306",
				},
				InstanceCount:          1,
				InstanceType:           "Standard_B2s",
				LBBackendAddressPoolID: createLBOutput.BackendAddressPoolIDs["backend-pool-master"],
				LBInboundNATPoolID:     createLBOutput.InboundNATPoolIDs["ssh-inbound-nat-pool"],
				Location:               input.Location,
				Name:                   input.ClusterName + "-vmss-master",
				NetworkSecurityGroupID: masterNSGOutput.NetworkSecurityGroupID,
				SSHPublicKey:           input.SSHPublicKey,
				SubnetID:               createVnetOutput.SubnetIDs[input.ClusterName+"-subnet-master"],
				Zones:                  []string{"1", "2", "3"},
			},
		}

		err := workflow.ExecuteActivity(ctx, CreateVMSSActivityName, activityInput).Get(ctx, &masterVMSSOutput)
		if err != nil {
			return err
		}
	}
	{
		activityInput := AssignRoleActivityInput{
			OrganizationID:    input.OrganizationID,
			SecretID:          input.SecretID,
			ClusterName:       input.ClusterName,
			ResourceGroupName: input.ResourceGroupName,
			PrincipalID:       masterVMSSOutput.PrincipalID,
			RoleName:          "Contributor",
			Name:              uuid.Must(uuid.NewV1()).String(),
		}
		err := workflow.ExecuteActivity(ctx, AssignRoleActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	// Create worker scale set
	var workerVMSSOutput CreateVMSSActivityOutput
	{
		activityInput := CreateVMSSActivityInput{
			OrganizationID:         input.OrganizationID,
			SecretID:               input.SecretID,
			ClusterName:            input.ClusterName,
			ClusterID:              input.ClusterID,
			ResourceGroupName:      input.ResourceGroupName,
			UserDataScriptTemplate: WorkerUserDataScriptTemplate,
			UserDataScriptParams: map[string]string{
				"": "",
			},
			ScaleSet: VirtualMachineScaleSet{
				AdminUsername: "azureuser",
				Image: Image{
					Offer:     "CentOS-CI",
					Publisher: "OpenLogic",
					SKU:       "7-CI",
					Version:   "7.6.20190306",
				},
				InstanceCount:          1,
				InstanceType:           "Standard_B2s",
				Location:               input.Location,
				Name:                   input.ClusterName + "-vmss-worker",
				NetworkSecurityGroupID: workerNSGOutput.NetworkSecurityGroupID,
				SSHPublicKey:           input.SSHPublicKey,
				SubnetID:               createVnetOutput.SubnetIDs[input.ClusterName+"-subnet-worker"],
				Zones:                  []string{"1", "2", "3"},
			},
		}

		err := workflow.ExecuteActivity(ctx, CreateVMSSActivityName, activityInput).Get(ctx, &workerVMSSOutput)
		if err != nil {
			return err
		}
	}
	{
		activityInput := AssignRoleActivityInput{
			OrganizationID:    input.OrganizationID,
			SecretID:          input.SecretID,
			ClusterName:       input.ClusterName,
			ResourceGroupName: input.ResourceGroupName,
			PrincipalID:       workerVMSSOutput.PrincipalID,
			RoleName:          "Contributor",
			Name:              uuid.Must(uuid.NewV1()).String(),
		}
		err := workflow.ExecuteActivity(ctx, AssignRoleActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}
	return nil
}
