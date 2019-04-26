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

	"github.com/banzaicloud/pipeline/cluster"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"go.uber.org/cadence/workflow"
)

const CreateClusterWorkflowName = "pke-azure-create-cluster"

// CreateClusterWorkflowInput
type CreateClusterWorkflowInput struct {
	ClusterID                       uint
	ClusterName                     string
	OrganizationID                  uint
	ResourceGroupName               string
	SecretID                        string
	VirtualNetworkTemplate          VirtualNetworkTemplate
	LoadBalancerTemplate            LoadBalancerTemplate
	PublicIPAddress                 PublicIPAddress
	RoleAssignmentTemplates         []RoleAssignmentTemplate
	RouteTable                      RouteTable
	SecurityGroups                  []SecurityGroup
	VirtualMachineScaleSetTemplates []VirtualMachineScaleSetTemplate
	PostHooks                       pkgCluster.PostHooks
}

type VirtualNetworkTemplate struct {
	Name     string
	CIDRs    []string
	Location string
	Subnets  []SubnetTemplate
}

type SubnetTemplate struct {
	Name                     string
	CIDR                     string
	NetworkSecurityGroupName string
}

type vnetFactory struct {
	template VirtualNetworkTemplate
}

func (f vnetFactory) Make(routeTableIDProvider IDProvider, securityGroupIDProvider IDByNameProvider) VirtualNetwork {
	subnets := make([]Subnet, len(f.template.Subnets))
	routeTableID := routeTableIDProvider.Get()
	for i, s := range f.template.Subnets {
		subnets[i] = Subnet{
			Name:                   s.Name,
			CIDR:                   s.CIDR,
			NetworkSecurityGroupID: securityGroupIDProvider.Get(s.NetworkSecurityGroupName),
			RouteTableID:           routeTableID,
		}
	}
	return VirtualNetwork{
		Name:     f.template.Name,
		CIDRs:    f.template.CIDRs,
		Location: f.template.Location,
		Subnets:  subnets,
	}
}

type LoadBalancerTemplate struct {
	Name                   string
	Location               string
	SKU                    string
	BackendAddressPoolName string
	InboundNATPoolName     string
}

type lbFactory struct {
	template LoadBalancerTemplate
}

func (f lbFactory) Make(publicIPAddressIDProvider IDProvider) LoadBalancer {
	bap := BackendAddressPool{
		Name: f.template.BackendAddressPoolName,
	}
	fic := FrontendIPConfiguration{
		Name:              "frontend-ip-config",
		PublicIPAddressID: publicIPAddressIDProvider.Get(),
	}
	probe := Probe{
		Name:     "api-server-probe",
		Port:     6443,
		Protocol: "Tcp",
	}
	return LoadBalancer{
		Name:     f.template.Name,
		Location: f.template.Location,
		SKU:      f.template.SKU,
		BackendAddressPools: []BackendAddressPool{
			bap,
		},
		FrontendIPConfigurations: []FrontendIPConfiguration{
			fic,
		},
		InboundNATPools: []InboundNATPool{
			{
				Name:                   f.template.InboundNATPoolName,
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
	}
}

type VirtualMachineScaleSetTemplate struct {
	AdminUsername            string
	Image                    Image
	InstanceCount            uint
	InstanceType             string
	BackendAddressPoolName   string
	InboundNATPoolName       string
	Location                 string
	Name                     string
	NetworkSecurityGroupName string
	SSHPublicKey             string
	SubnetName               string
	UserDataScriptParams     map[string]string
	UserDataScriptTemplate   string
	Zones                    []string
}

type vmssFactory struct {
	templates []VirtualMachineScaleSetTemplate
}

func (f vmssFactory) Make(
	backendAddressPoolIDProvider IDByNameProvider,
	inboundNATPoolIDProvider IDByNameProvider,
	publicIPAddressProvider IPAddressProvider,
	securityGroupIDProvider IDByNameProvider,
	subnetIDProvider IDByNameProvider,
) []VirtualMachineScaleSet {
	publicIPAddress := publicIPAddressProvider.Get()
	sss := make([]VirtualMachineScaleSet, len(f.templates))
	for i, t := range f.templates {
		t.UserDataScriptParams["PublicIPAddress"] = publicIPAddress
		sss[i] = VirtualMachineScaleSet{
			AdminUsername:          t.AdminUsername,
			Image:                  t.Image,
			InstanceCount:          int64(t.InstanceCount),
			InstanceType:           t.InstanceType,
			LBBackendAddressPoolID: backendAddressPoolIDProvider.Get(t.BackendAddressPoolName),
			LBInboundNATPoolID:     inboundNATPoolIDProvider.Get(t.InboundNATPoolName),
			Location:               t.Location,
			Name:                   t.Name,
			NetworkSecurityGroupID: securityGroupIDProvider.Get(t.NetworkSecurityGroupName),
			SSHPublicKey:           t.SSHPublicKey,
			SubnetID:               subnetIDProvider.Get(t.SubnetName),
			UserDataScriptTemplate: t.UserDataScriptTemplate,
			UserDataScriptParams:   t.UserDataScriptParams,
			Zones:                  t.Zones,
		}
	}
	return sss
}

type RoleAssignmentTemplate struct {
	Name     string
	VMSSName string
	RoleName string
}

type roleAssignmentsFactory struct {
	templates []RoleAssignmentTemplate
}

func (f roleAssignmentsFactory) Make(vmssPrincipalIDProvider IDByNameProvider) []RoleAssignment {
	ras := make([]RoleAssignment, len(f.templates))
	for i, ra := range f.templates {
		ras[i] = RoleAssignment{
			Name:        ra.Name,
			PrincipalID: vmssPrincipalIDProvider.Get(ra.VMSSName),
			RoleName:    ra.RoleName,
		}
	}
	return ras
}

func CreateClusterWorkflow(ctx workflow.Context, input CreateClusterWorkflowInput) error {

	cwo := workflow.ChildWorkflowOptions{
		ExecutionStartToCloseTimeout: time.Minute,
	}
	ctx = workflow.WithChildOptions(ctx, cwo)

	infraInput := CreateAzureInfrastructureWorkflowInput{
		OrganizationID:    input.OrganizationID,
		ClusterID:         input.ClusterID,
		ClusterName:       input.ClusterName,
		SecretID:          input.SecretID,
		ResourceGroupName: input.ResourceGroupName,
		LoadBalancer: lbFactory{
			template: input.LoadBalancerTemplate,
		},
		PublicIPAddress: input.PublicIPAddress,
		RoleAssignments: roleAssignmentsFactory{
			templates: input.RoleAssignmentTemplates,
		},
		RouteTable: input.RouteTable,
		ScaleSets: vmssFactory{
			templates: input.VirtualMachineScaleSetTemplates,
		},
		SecurityGroups: input.SecurityGroups,
		VirtualNetwork: vnetFactory{
			template: input.VirtualNetworkTemplate,
		},
	}
	err := workflow.ExecuteChildWorkflow(ctx, CreateInfraWorkflowName, infraInput).Get(ctx, nil)
	if err != nil {
		return err
	}

	postHookWorkflowInput := cluster.RunPostHooksWorkflowInput{
		ClusterID: input.ClusterID,
		PostHooks: cluster.BuildWorkflowPostHookFunctions(input.PostHooks, true),
	}

	err = workflow.ExecuteChildWorkflow(ctx, cluster.RunPostHooksWorkflowName, postHookWorkflowInput).Get(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}
