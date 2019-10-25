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
	"net"
	"strconv"
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence/workflow"

	intPKE "github.com/banzaicloud/pipeline/internal/pke"
	intPKEWorkflow "github.com/banzaicloud/pipeline/internal/pke/workflow"
)

const CreateInfraWorkflowName = "pke-azure-create-infra"

type CreateAzureInfrastructureWorkflowInput struct {
	OrganizationID    uint
	ClusterID         uint
	ClusterName       string
	SecretID          string
	ResourceGroupName string

	LoadBalancer    LoadBalancerTemplate
	PublicIPAddress PublicIPAddress
	RoleAssignments []RoleAssignmentTemplate
	RouteTable      RouteTable
	ScaleSets       []VirtualMachineScaleSetTemplate
	SecurityGroups  []SecurityGroup
	VirtualNetwork  VirtualNetworkTemplate
	HTTPProxy       intPKE.HTTPProxy
}

type LoadBalancerTemplate struct {
	Name                           string
	Location                       string
	SKU                            string
	BackendAddressPoolName         string
	OutboundBackendAddressPoolName string
	InboundNATPoolName             string
}

func (t LoadBalancerTemplate) Render(publicIPAddressIDProvider ResourceIDProvider) LoadBalancer {
	bap := BackendAddressPool{
		Name: t.BackendAddressPoolName,
	}
	obap := BackendAddressPool{
		Name: t.OutboundBackendAddressPoolName,
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
		Name:     t.Name,
		Location: t.Location,
		SKU:      t.SKU,
		BackendAddressPools: []BackendAddressPool{
			bap,
			obap,
		},
		FrontendIPConfigurations: []FrontendIPConfiguration{
			fic,
		},
		InboundNATPools: []InboundNATPool{
			{
				Name:                   t.InboundNATPoolName,
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
				DisableOutboundSNAT: true,
				FrontendIPConfig:    &fic,
				FrontendPort:        6443,
				Probe:               &probe,
				Protocol:            "Tcp",
			},
		},
		OutboundRules: []OutboundRule{
			{
				Name:               "outbound-nat-rule",
				BackendAddressPool: &obap,
				FrontendIPConfigs:  []*FrontendIPConfiguration{&fic},
			},
		},
		Probes: []Probe{
			probe,
		},
	}
}

type RoleAssignmentTemplate struct {
	Name     string
	VMSSName string
	RoleName string
}

func (t RoleAssignmentTemplate) Render(vmssPrincipalIDProvider ResourceIDByNameProvider) RoleAssignment {
	return RoleAssignment{
		Name:        t.Name,
		PrincipalID: vmssPrincipalIDProvider.Get(t.VMSSName),
		RoleName:    t.RoleName,
	}
}

type VirtualMachineScaleSetTemplate struct {
	AdminUsername                string
	Image                        Image
	InstanceCount                uint
	InstanceType                 string
	BackendAddressPoolName       string
	OutputBackendAddressPoolName string
	InboundNATPoolName           string
	Location                     string
	Name                         string
	NetworkSecurityGroupName     string
	NodePoolName                 string
	SSHPublicKey                 string
	SubnetName                   string
	UserDataScriptParams         map[string]string
	UserDataScriptTemplate       string
	Zones                        []string
}

func (t VirtualMachineScaleSetTemplate) Render(
	backendAddressPoolIDProvider ResourceIDByNameProvider,
	inboundNATPoolIDProvider ResourceIDByNameProvider,
	publicIPAddressProvider IPAddressProvider,
	securityGroupIDProvider ResourceIDByNameProvider,
	subnetIDProvider ResourceIDByNameProvider,
) VirtualMachineScaleSet {
	t.UserDataScriptParams["PublicAddress"] = publicIPAddressProvider.Get()
	return VirtualMachineScaleSet{
		AdminUsername: t.AdminUsername,
		Image:         t.Image,
		InstanceCount: int64(t.InstanceCount),
		InstanceType:  t.InstanceType,
		LBBackendAddressPoolIDs: []string{
			backendAddressPoolIDProvider.Get(t.BackendAddressPoolName),
			backendAddressPoolIDProvider.Get(t.OutputBackendAddressPoolName),
		},
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

type SubnetTemplate struct {
	Name                     string
	CIDR                     string
	NetworkSecurityGroupName string
	RouteTableName           string
}

func (t SubnetTemplate) Render(routeTableIDProvider ResourceIDByNameProvider, securityGroupIDProvider ResourceIDByNameProvider) Subnet {
	return Subnet{
		Name:                   t.Name,
		CIDR:                   t.CIDR,
		NetworkSecurityGroupID: securityGroupIDProvider.Get(t.NetworkSecurityGroupName),
		RouteTableID:           routeTableIDProvider.Get(t.RouteTableName),
	}
}

type VirtualNetworkTemplate struct {
	Name     string
	CIDRs    []string
	Location string
	Subnets  []SubnetTemplate
}

func (t VirtualNetworkTemplate) Render(routeTableIDProvider ResourceIDByNameProvider, securityGroupIDProvider ResourceIDByNameProvider) VirtualNetwork {
	subnets := make([]Subnet, len(t.Subnets))
	for i, s := range t.Subnets {
		subnets[i] = s.Render(routeTableIDProvider, securityGroupIDProvider)
	}
	return VirtualNetwork{
		Name:     t.Name,
		CIDRs:    t.CIDRs,
		Location: t.Location,
		Subnets:  subnets,
	}
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

func (p routeTableIDProvider) Get(name string) string {
	if p.RouteTableName == name {
		return p.RouteTableID
	}
	return ""
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
		futures := make([]workflow.Future, len(input.SecurityGroups))

		for i, sg := range input.SecurityGroups {
			activityInput := CreateNSGActivityInput{
				OrganizationID:    input.OrganizationID,
				SecretID:          input.SecretID,
				ClusterName:       input.ClusterName,
				ResourceGroupName: input.ResourceGroupName,
				SecurityGroup:     sg,
			}

			futures[i] = workflow.ExecuteActivity(ctx, CreateNSGActivityName, activityInput)
		}

		errs := make([]error, len(futures))

		for i, future := range futures {
			var activityOutput CreateNSGActivityOutput

			errs[i] = future.Get(ctx, &activityOutput)

			createNSGActivityOutputs[input.SecurityGroups[i].Name] = activityOutput
		}

		if err := errors.Combine(errs...); err != nil {
			return err
		}
	}

	// Create route table
	var createRouteTableActivityOutput CreateRouteTableActivityOutput
	if input.RouteTable.ID == "" {
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
	} else {
		createRouteTableActivityOutput.RouteTableID = input.RouteTable.ID
		createRouteTableActivityOutput.RouteTableName = input.RouteTable.Name
	}

	// Create virtual network and subnets
	var createVnetOutput CreateVnetActivityOutput
	{
		activityInput := CreateVnetActivityInput{
			ResourceGroupName: input.ResourceGroupName,
			OrganizationID:    input.OrganizationID,
			ClusterName:       input.ClusterName,
			SecretID:          input.SecretID,
			VirtualNetwork:    input.VirtualNetwork.Render(routeTableIDProvider(createRouteTableActivityOutput), mapSecurityGroupIDProvider(createNSGActivityOutputs)),
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
			LoadBalancer:      input.LoadBalancer.Render(publicIPAddressIDProvider(createPublicIPActivityOutput)),
		}
		if err := workflow.ExecuteActivity(ctx, CreateLoadBalancerActivityName, activityInput).Get(ctx, &createLBActivityOutput); err != nil {
			return err
		}
	}

	var httpProxy intPKEWorkflow.HTTPProxy
	{
		activityInput := intPKEWorkflow.AssembleHTTPProxySettingsActivityInput{
			OrganizationID:     input.OrganizationID,
			HTTPProxyHostPort:  getHostPort(input.HTTPProxy.HTTP),
			HTTPProxySecretID:  input.HTTPProxy.HTTP.SecretID,
			HTTPSProxyHostPort: getHostPort(input.HTTPProxy.HTTPS),
			HTTPSProxySecretID: input.HTTPProxy.HTTPS.SecretID,
		}
		var output intPKEWorkflow.AssembleHTTPProxySettingsActivityOutput
		if err := workflow.ExecuteActivity(ctx, intPKEWorkflow.AssembleHTTPProxySettingsActivityName, activityInput).Get(ctx, &output); err != nil {
			return err
		}
		httpProxy = output.Settings
	}

	// Create scale sets
	createVMSSActivityOutputs := make(map[string]CreateVMSSActivityOutput)
	{
		bapIDProvider := backendAddressPoolIDProvider(createLBActivityOutput)
		inpIDProvider := inboundNATPoolIDProvider(createLBActivityOutput)
		pipIDProvider := publicIPAddressIPAddressProvider(createPublicIPActivityOutput)
		nsgIDProvider := mapSecurityGroupIDProvider(createNSGActivityOutputs)
		subnetIDProvider := subnetIDProvider(createVnetOutput)

		futures := make([]workflow.Future, len(input.ScaleSets))

		for i, vmssTemplate := range input.ScaleSets {
			activityInput := CreateVMSSActivityInput{
				OrganizationID:    input.OrganizationID,
				SecretID:          input.SecretID,
				ClusterID:         input.ClusterID,
				ClusterName:       input.ClusterName,
				ResourceGroupName: input.ResourceGroupName,
				ScaleSet:          vmssTemplate.Render(bapIDProvider, inpIDProvider, pipIDProvider, nsgIDProvider, subnetIDProvider),
				HTTPProxy:         httpProxy,
			}
			futures[i] = workflow.ExecuteActivity(ctx, CreateVMSSActivityName, activityInput)
		}

		errs := make([]error, len(futures))

		for i, future := range futures {
			var activityOutput CreateVMSSActivityOutput

			errs[i] = errors.WrapIff(future.Get(ctx, &activityOutput), "creating scaling set %q", input.ScaleSets[i].Name)

			createVMSSActivityOutputs[input.ScaleSets[i].Name] = activityOutput
		}

		if err := errors.Combine(errs...); err != nil {
			return err
		}
	}

	// Create role assignments
	{
		vmssPrincipalIDProvider := mapVMSSPrincipalIDProvider(createVMSSActivityOutputs)

		futures := make([]workflow.Future, len(input.RoleAssignments))

		for i, t := range input.RoleAssignments {
			activityInput := AssignRoleActivityInput{
				OrganizationID:    input.OrganizationID,
				SecretID:          input.SecretID,
				ClusterName:       input.ClusterName,
				ResourceGroupName: input.ResourceGroupName,
				RoleAssignment:    t.Render(vmssPrincipalIDProvider),
			}
			futures[i] = workflow.ExecuteActivity(ctx, AssignRoleActivityName, activityInput)
		}

		errs := make([]error, len(futures))

		for i, future := range futures {
			errs[i] = future.Get(ctx, nil)
		}

		if err := errors.Combine(errs...); err != nil {
			return err
		}
	}

	return nil
}

func getHostPort(o intPKE.HTTPProxyOptions) string {
	if o.Host == "" {
		return ""
	}
	if o.Port == 0 {
		return o.Host
	}
	return net.JoinHostPort(o.Host, strconv.FormatUint(uint64(o.Port), 10))
}
