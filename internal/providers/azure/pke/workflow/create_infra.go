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

	"github.com/goph/emperror"
	"go.uber.org/cadence/workflow"
)

const CreateInfraWorkflowName = "pke-azure-create-infra"

type CreateAzureInfrastructureWorkflowInput struct {
	OrganizationID    uint
	ClusterID         uint
	ClusterName       string
	SecretID          string
	ResourceGroupName string

	LoadBalancer    LoadBalancerFactory
	PublicIPAddress PublicIPAddress
	RoleAssignments RoleAssignmentsFactory
	RouteTable      RouteTable
	ScaleSets       VirtualMachineScaleSetsFactory
	SecurityGroups  []SecurityGroup
	VirtualNetwork  VirtualNetworkFactory
}

type LoadBalancerFactory struct {
	Template LoadBalancerTemplate
}

type LoadBalancerTemplate struct {
	Name                   string
	Location               string
	SKU                    string
	BackendAddressPoolName string
	InboundNATPoolName     string
}

func (f LoadBalancerFactory) Make(publicIPAddressIDProvider IDProvider) LoadBalancer {
	bap := BackendAddressPool{
		Name: f.Template.BackendAddressPoolName,
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
		Name:     f.Template.Name,
		Location: f.Template.Location,
		SKU:      f.Template.SKU,
		BackendAddressPools: []BackendAddressPool{
			bap,
		},
		FrontendIPConfigurations: []FrontendIPConfiguration{
			fic,
		},
		InboundNATPools: []InboundNATPool{
			{
				Name:                   f.Template.InboundNATPoolName,
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

type RoleAssignmentsFactory struct {
	Templates []RoleAssignmentTemplate
}

type RoleAssignmentTemplate struct {
	Name     string
	VMSSName string
	RoleName string
}

func (f RoleAssignmentsFactory) Make(vmssPrincipalIDProvider IDByNameProvider) []RoleAssignment {
	ras := make([]RoleAssignment, len(f.Templates))
	for i, ra := range f.Templates {
		ras[i] = RoleAssignment{
			Name:        ra.Name,
			PrincipalID: vmssPrincipalIDProvider.Get(ra.VMSSName),
			RoleName:    ra.RoleName,
		}
	}
	return ras
}

type VirtualMachineScaleSetsFactory struct {
	Templates []VirtualMachineScaleSetTemplate
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

func (f VirtualMachineScaleSetsFactory) Make(
	backendAddressPoolIDProvider IDByNameProvider,
	inboundNATPoolIDProvider IDByNameProvider,
	publicIPAddressProvider IPAddressProvider,
	securityGroupIDProvider IDByNameProvider,
	subnetIDProvider IDByNameProvider,
) []VirtualMachineScaleSet {
	publicIPAddress := publicIPAddressProvider.Get()
	sss := make([]VirtualMachineScaleSet, len(f.Templates))
	for i, t := range f.Templates {
		t.UserDataScriptParams["PublicAddress"] = publicIPAddress
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

type VirtualNetworkFactory struct {
	Template VirtualNetworkTemplate
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

func (f VirtualNetworkFactory) Make(routeTableIDProvider IDProvider, securityGroupIDProvider IDByNameProvider) VirtualNetwork {
	subnets := make([]Subnet, len(f.Template.Subnets))
	routeTableID := routeTableIDProvider.Get()
	for i, s := range f.Template.Subnets {
		subnets[i] = Subnet{
			Name:                   s.Name,
			CIDR:                   s.CIDR,
			NetworkSecurityGroupID: securityGroupIDProvider.Get(s.NetworkSecurityGroupName),
			RouteTableID:           routeTableID,
		}
	}
	return VirtualNetwork{
		Name:     f.Template.Name,
		CIDRs:    f.Template.CIDRs,
		Location: f.Template.Location,
		Subnets:  subnets,
	}
}

type IDProvider interface {
	Get() string
}

type IDByNameProvider interface {
	Get(name string) string
}

type IPAddressProvider interface {
	Get() string
}

type backendAddressPoolIDProvider CreateLoadBalancerActivityOutput

func (p backendAddressPoolIDProvider) Get(name string) string {
	return p.BackendAddressPoolIDs[name]
}

type inboundNATPoolIDProvider CreateLoadBalancerActivityOutput

func (p inboundNATPoolIDProvider) Get(name string) string {
	return p.InboundNATPoolIDs[name]
}

type subnetIDProvider CreateVnetActivityOutput

func (p subnetIDProvider) Get(name string) string {
	return p.SubnetIDs[name]
}

type publicIPAddressIPAddressProvider CreatePublicIPActivityOutput

func (p publicIPAddressIPAddressProvider) Get() string {
	return p.PublicIPAddress
}

type publicIPAddressIDProvider CreatePublicIPActivityOutput

func (p publicIPAddressIDProvider) Get() string {
	return p.PublicIPAddressID
}

type routeTableIDProvider CreateRouteTableActivityOutput

func (p routeTableIDProvider) Get() string {
	return p.RouteTableID
}

type mapSecurityGroupIDProvider map[string]CreateNSGActivityOutput

func (p mapSecurityGroupIDProvider) Get(name string) string {
	return p[name].NetworkSecurityGroupID
}

type mapVMSSPrincipalIDProvider map[string]CreateVMSSActivityOutput

func (p mapVMSSPrincipalIDProvider) Get(name string) string {
	return p[name].PrincipalID
}

func CreateInfrastructureWorkflow(ctx workflow.Context, input CreateAzureInfrastructureWorkflowInput) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    15 * time.Minute,
		ScheduleToCloseTimeout: 20 * time.Minute,
		WaitForCancellation:    true,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	// Create network security groups
	createNSGActivityOutputs := make(map[string]CreateNSGActivityOutput)
	{
		futures := make(map[string]workflow.Future, len(input.SecurityGroups))
		for _, sg := range input.SecurityGroups {
			activityInput := CreateNSGActivityInput{
				OrganizationID:    input.OrganizationID,
				SecretID:          input.SecretID,
				ClusterName:       input.ClusterName,
				ResourceGroupName: input.ResourceGroupName,
				SecurityGroup:     sg,
			}
			futures[sg.Name] = workflow.ExecuteActivity(ctx, CreateNSGActivityName, activityInput)
		}
		for name, future := range futures {
			var activityOutput CreateNSGActivityOutput
			if err := future.Get(ctx, &activityOutput); err != nil {
				return err
			}
			createNSGActivityOutputs[name] = activityOutput
		}
	}

	// Create route table
	var createRouteTableActivityOutput CreateRouteTableActivityOutput
	{
		activityInput := CreateRouteTableActivityInput{
			OrganizationID:    input.OrganizationID,
			SecretID:          input.SecretID,
			ClusterName:       input.ClusterName,
			ResourceGroupName: input.ResourceGroupName,
			RouteTable:        input.RouteTable,
		}
		if err := workflow.ExecuteActivity(ctx, CreateRouteTableActivityName, activityInput).Get(ctx, &createRouteTableActivityOutput); err != nil {
			return err
		}
	}

	// Create virtual network and subnets
	var createVnetOutput CreateVnetActivityOutput
	{
		activityInput := CreateVnetActivityInput{
			ResourceGroupName: input.ResourceGroupName,
			OrganizationID:    input.OrganizationID,
			ClusterName:       input.ClusterName,
			SecretID:          input.SecretID,
			VirtualNetwork:    input.VirtualNetwork.Make(routeTableIDProvider(createRouteTableActivityOutput), mapSecurityGroupIDProvider(createNSGActivityOutputs)),
		}
		if err := workflow.ExecuteActivity(ctx, CreateVnetActivityName, activityInput).Get(ctx, &createVnetOutput); err != nil {
			return err
		}
	}

	// Create PublicIP
	var createPublicIPActivityOutput CreatePublicIPActivityOutput
	{
		activityInput := CreatePublicIPActivityInput{
			OrganizationID:    input.OrganizationID,
			SecretID:          input.SecretID,
			ClusterName:       input.ClusterName,
			ResourceGroupName: input.ResourceGroupName,
			PublicIPAddress:   input.PublicIPAddress,
		}
		if err := workflow.ExecuteActivity(ctx, CreatePublicIPActivityName, activityInput).Get(ctx, &createPublicIPActivityOutput); err != nil {
			return err
		}
	}

	// Create load balancer
	var createLBActivityOutput CreateLoadBalancerActivityOutput
	{
		activityInput := CreateLoadBalancerActivityInput{
			OrganizationID:    input.OrganizationID,
			SecretID:          input.SecretID,
			ClusterName:       input.ClusterName,
			ResourceGroupName: input.ResourceGroupName,
			LoadBalancer:      input.LoadBalancer.Make(publicIPAddressIDProvider(createPublicIPActivityOutput)),
		}
		if err := workflow.ExecuteActivity(ctx, CreateLoadBalancerActivityName, activityInput).Get(ctx, &createLBActivityOutput); err != nil {
			return err
		}
	}

	// Create scale sets
	createVMSSActivityOutputs := make(map[string]CreateVMSSActivityOutput)
	{
		scaleSets := input.ScaleSets.Make(
			backendAddressPoolIDProvider(createLBActivityOutput),
			inboundNATPoolIDProvider(createLBActivityOutput),
			publicIPAddressIPAddressProvider(createPublicIPActivityOutput),
			mapSecurityGroupIDProvider(createNSGActivityOutputs),
			subnetIDProvider(createVnetOutput),
		)
		futures := make(map[string]workflow.Future, len(scaleSets))
		for _, vmss := range scaleSets {
			activityInput := CreateVMSSActivityInput{
				OrganizationID:    input.OrganizationID,
				SecretID:          input.SecretID,
				ClusterID:         input.ClusterID,
				ClusterName:       input.ClusterName,
				ResourceGroupName: input.ResourceGroupName,
				ScaleSet:          vmss,
			}
			futures[vmss.Name] = workflow.ExecuteActivity(ctx, CreateVMSSActivityName, activityInput)
		}

		for name, future := range futures {
			var activityOutput CreateVMSSActivityOutput
			if err := future.Get(ctx, &activityOutput); err != nil {
				return emperror.Wrapf(err, "creating scaling set %q", name)
			}
			createVMSSActivityOutputs[name] = activityOutput
		}
	}

	// Create role assignments
	{
		roleAssignments := input.RoleAssignments.Make(mapVMSSPrincipalIDProvider(createVMSSActivityOutputs))
		futures := make([]workflow.Future, len(roleAssignments))
		for i, ra := range roleAssignments {
			activityInput := AssignRoleActivityInput{
				OrganizationID:    input.OrganizationID,
				SecretID:          input.SecretID,
				ClusterName:       input.ClusterName,
				ResourceGroupName: input.ResourceGroupName,
				RoleAssignment:    ra,
			}
			futures[i] = workflow.ExecuteActivity(ctx, AssignRoleActivityName, activityInput)
		}
		for _, future := range futures {
			if err := future.Get(ctx, nil); err != nil {
				return err
			}
		}
	}

	return nil
}
