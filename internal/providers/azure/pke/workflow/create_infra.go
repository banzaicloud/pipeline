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
	"strings"
	"time"

	"emperror.dev/errors"

	pkgPke "github.com/banzaicloud/pipeline/pkg/cluster/pke"

	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"

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

	LoadBalancers   []LoadBalancerTemplate
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
	SubnetName                     string
	PublicIPAddressName            string
}

func (t LoadBalancerTemplate) Render(publicIPAddressIDProvider ResourceIDByNameProvider, subnetIDProvider ResourceIDByNameProvider) LoadBalancer {
	var backendAddressPools []BackendAddressPool
	var bap, obap *BackendAddressPool

	if t.BackendAddressPoolName != "" {
		bap = &BackendAddressPool{
			Name: t.BackendAddressPoolName,
		}
		backendAddressPools = append(backendAddressPools, *bap)
	}

	if t.OutboundBackendAddressPoolName != "" {
		obap = &BackendAddressPool{
			Name: t.OutboundBackendAddressPoolName,
		}
		backendAddressPools = append(backendAddressPools, *obap)
	}

	fic := FrontendIPConfiguration{
		Name:              pke.GetFrontEndIPConfigName(),
		PublicIPAddressID: publicIPAddressIDProvider.Get(t.PublicIPAddressName),
		SubnetID:          subnetIDProvider.Get(t.SubnetName),
	}

	var inboundNATPools []InboundNATPool
	if t.InboundNATPoolName != "" {
		inboundNATPools = append(inboundNATPools, InboundNATPool{
			Name:                   t.InboundNATPoolName,
			BackendPort:            22,
			FrontendIPConfig:       &fic,
			FrontendPortRangeEnd:   50100,
			FrontendPortRangeStart: 50000,
			Protocol:               "Tcp",
		})
	}

	var probes []Probe
	var lbRules []LoadBalancingRule

	if bap != nil {
		apiServerProbe := Probe{
			Name:     "api-server-probe",
			Port:     6443,
			Protocol: "Tcp",
		}

		apiServerLBRule := LoadBalancingRule{
			Name:                pke.GetApiServerLBRuleName(),
			BackendAddressPool:  bap,
			BackendPort:         6443,
			DisableOutboundSNAT: true,
			FrontendIPConfig:    &fic,
			FrontendPort:        6443,
			Probe:               &apiServerProbe,
			Protocol:            "Tcp",
		}

		probes = append(probes, apiServerProbe)
		lbRules = append(lbRules, apiServerLBRule)
	}

	var lbOutboundRules []OutboundRule
	if obap != nil {
		lbOutboundRules = append(lbOutboundRules, OutboundRule{
			Name:               "outbound-nat-rule",
			BackendAddressPool: obap,
			FrontendIPConfigs:  []*FrontendIPConfiguration{&fic},
		})
	}

	return LoadBalancer{
		Name:                t.Name,
		Location:            t.Location,
		SKU:                 t.SKU,
		BackendAddressPools: backendAddressPools,
		FrontendIPConfigurations: []FrontendIPConfiguration{
			fic,
		},
		InboundNATPools:    inboundNATPools,
		LoadBalancingRules: lbRules,
		OutboundRules:      lbOutboundRules,
		Probes:             probes,
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
	Role                         pkgPke.Role
}

func (t VirtualMachineScaleSetTemplate) Render(
	backendAddressPoolIDProviders []ResourceIDByNameProvider,
	inboundNATPoolIDProviders []ResourceIDByNameProvider,
	apiServerAddressProvider IPAddressProvider,
	apiServerCertSansProvider ConstantResourceIDProvider,
	securityGroupIDProvider ResourceIDByNameProvider,
	subnetIDProvider ResourceIDByNameProvider,
) VirtualMachineScaleSet {
	var backendAddressPoolIDs []string
	for _, resourceIDProvider := range backendAddressPoolIDProviders {
		backendAddressPoolIDs = append(backendAddressPoolIDs, resourceIDProvider.Get(t.BackendAddressPoolName), resourceIDProvider.Get(t.OutputBackendAddressPoolName))
	}

	var inboundNATPoolIDs []string
	for _, resourceIDProvider := range inboundNATPoolIDProviders {
		inboundNATPoolIDs = append(inboundNATPoolIDs, resourceIDProvider.Get(t.InboundNATPoolName))
	}

	t.UserDataScriptParams["ApiServerAddress"] = apiServerAddressProvider.Get()
	t.UserDataScriptParams["ApiServerCertSans"] = apiServerCertSansProvider.Get()
	return VirtualMachineScaleSet{
		AdminUsername:           t.AdminUsername,
		Image:                   t.Image,
		InstanceCount:           int64(t.InstanceCount),
		InstanceType:            t.InstanceType,
		LBBackendAddressPoolIDs: backendAddressPoolIDs,
		LBInboundNATPoolIDs:     inboundNATPoolIDs,
		Location:                t.Location,
		Name:                    t.Name,
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
		Name: t.Name,
		CIDR: t.CIDR,
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

func (p publicIPAddressIDProvider) Get(name string) string {
	if p.Name == name {
		return p.PublicIPAddressID
	}
	return ""
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
	subnetIDProvider := subnetIDProvider(createVnetOutput)

	// Create PublicIP
	var createPublicIPActivityOutput CreatePublicIPActivityOutput
	if input.PublicIPAddress.Name != "" {
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

	// Create load balancers
	var createLBActivityOutputs []CreateLoadBalancerActivityOutput
	futures := make([]workflow.Future, len(input.LoadBalancers))

	for i, lb := range input.LoadBalancers {
		activityInput := CreateLoadBalancerActivityInput{
			OrganizationID:    input.OrganizationID,
			SecretID:          input.SecretID,
			ClusterName:       input.ClusterName,
			ResourceGroupName: input.ResourceGroupName,
			LoadBalancer:      lb.Render(publicIPAddressIDProvider(createPublicIPActivityOutput), subnetIDProvider),
		}

		futures[i] = workflow.ExecuteActivity(ctx, CreateLoadBalancerActivityName, activityInput)
	}

	errs := make([]error, len(futures))
	for i, future := range futures {
		var output CreateLoadBalancerActivityOutput
		errs[i] = future.Get(ctx, &output)

		if errs[i] == nil {
			createLBActivityOutputs = append(createLBActivityOutputs, output)
		}
	}

	if err := errors.Combine(errs...); err != nil {
		return err
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
		var apiServerPublicAddressProvider, apiServerPrivateAddressProvider IPAddressProvider
		apiServerCertSansMap := make(map[string]bool)

		if createPublicIPActivityOutput.PublicIPAddress != "" {
			apiServerCertSansMap[createPublicIPActivityOutput.PublicIPAddress] = true
			apiServerPublicAddressProvider = ConstantIPAddressProvider(createPublicIPActivityOutput.PublicIPAddress)
		}

		bapIDProviders := make([]ResourceIDByNameProvider, len(createLBActivityOutputs))
		inpIDProviders := make([]ResourceIDByNameProvider, len(createLBActivityOutputs))

		for i, createLBActivityOutput := range createLBActivityOutputs {
			bapIDProviders[i] = backendAddressPoolIDProvider(createLBActivityOutput)
			inpIDProviders[i] = inboundNATPoolIDProvider(createLBActivityOutput)

			if createLBActivityOutput.ApiServerPrivateAddress != "" {
				apiServerCertSansMap[createLBActivityOutput.ApiServerPrivateAddress] = true
				apiServerPrivateAddressProvider = ConstantIPAddressProvider(createLBActivityOutput.ApiServerPrivateAddress)
			}
		}

		var apiServerCertSans []string
		for certSan := range apiServerCertSansMap {
			apiServerCertSans = append(apiServerCertSans, certSan)
		}
		apiServerCertSansProvider := ConstantResourceIDProvider(strings.Join(apiServerCertSans, ","))

		nsgIDProvider := mapSecurityGroupIDProvider(createNSGActivityOutputs)

		futures := make([]workflow.Future, len(input.ScaleSets))
		for i, vmssTemplate := range input.ScaleSets {
			var apiServerAddressProvider IPAddressProvider

			if vmssTemplate.Role == pkgPke.RoleMaster {
				apiServerAddressProvider = ConstantIPAddressProvider("$PRIVATE_IP")
			} else {
				// worker nodes connect to API server through internal or public LB, internal has priority over public LB
				if apiServerPrivateAddressProvider != nil {
					apiServerAddressProvider = apiServerPrivateAddressProvider
				} else if apiServerPublicAddressProvider != nil {
					apiServerAddressProvider = apiServerPublicAddressProvider
				} else {
					return errors.New("no API server address available")
				}
			}

			activityInput := CreateVMSSActivityInput{
				OrganizationID:    input.OrganizationID,
				SecretID:          input.SecretID,
				ClusterID:         input.ClusterID,
				ClusterName:       input.ClusterName,
				ResourceGroupName: input.ResourceGroupName,
				ScaleSet:          vmssTemplate.Render(bapIDProviders, inpIDProviders, apiServerAddressProvider, apiServerCertSansProvider, nsgIDProvider, subnetIDProvider),
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
