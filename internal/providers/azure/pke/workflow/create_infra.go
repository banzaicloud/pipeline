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
	ClusterID         uint
	ClusterName       string
	SecretID          string
	Location          string
	ResourceGroupName string
	TenantID          string

	LoadBalancer    LoadBalancerFactory
	PublicIPAddress PublicIPAddress
	RoleAssignments RoleAssignmentsFactory
	RouteTable      RouteTable
	ScaleSets       VirtualMachineScaleSetsFactory
	SecurityGroups  []SecurityGroup
	VirtualNetwork  VirtualNetworkFactory
}

type LoadBalancerFactory interface {
	Make(publicIPAddressIDProvider IDProvider) LoadBalancer
}

type RoleAssignmentsFactory interface {
	Make(vmssPrincipalIDProvider IDByNameProvider) []RoleAssignment
}

type VirtualMachineScaleSetsFactory interface {
	Make(
		backendAddressPoolIDProvider IDByNameProvider,
		inboundNATPoolIDProvider IDByNameProvider,
		publicIPAddressProvider IPAddressProvider,
		securityGroupIDProvider IDByNameProvider,
		subnetIDProvider IDByNameProvider,
	) []VirtualMachineScaleSet
}

type VirtualNetworkFactory interface {
	Make(routeTableIDProvider IDProvider, securityGroupIDProvider IDByNameProvider) VirtualNetwork
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
		StartToCloseTimeout:    10 * time.Minute,
		ScheduleToCloseTimeout: 15 * time.Minute,
		WaitForCancellation:    true,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	// Create network security groups
	createNSGActivityOutputs := make(map[string]CreateNSGActivityOutput)
	for _, sg := range input.SecurityGroups {
		activityInput := CreateNSGActivityInput{
			OrganizationID:    input.OrganizationID,
			SecretID:          input.SecretID,
			ClusterName:       input.ClusterName,
			ResourceGroupName: input.ResourceGroupName,
			SecurityGroup:     sg,
		}
		var activityOutput CreateNSGActivityOutput
		if err := workflow.ExecuteActivity(ctx, CreateNSGActivityName, activityInput).Get(ctx, &activityOutput); err != nil {
			return err
		}
		createNSGActivityOutputs[sg.Name] = activityOutput
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
	for _, vmss := range input.ScaleSets.Make(
		backendAddressPoolIDProvider(createLBActivityOutput),
		inboundNATPoolIDProvider(createLBActivityOutput),
		publicIPAddressIPAddressProvider(createPublicIPActivityOutput),
		mapSecurityGroupIDProvider(createNSGActivityOutputs),
		subnetIDProvider(createVnetOutput),
	) {
		activityInput := CreateVMSSActivityInput{
			OrganizationID:    input.OrganizationID,
			SecretID:          input.SecretID,
			ClusterID:         input.ClusterID,
			ClusterName:       input.ClusterName,
			ResourceGroupName: input.ResourceGroupName,
			ScaleSet:          vmss,
		}
		var activityOutput CreateVMSSActivityOutput
		err := workflow.ExecuteActivity(ctx, CreateVMSSActivityName, activityInput).Get(ctx, &activityOutput)
		if err != nil {
			return err
		}
		createVMSSActivityOutputs[vmss.Name] = activityOutput
	}

	// Create role assignments
	for _, ra := range input.RoleAssignments.Make(mapVMSSPrincipalIDProvider(createVMSSActivityOutputs)) {
		activityInput := AssignRoleActivityInput{
			OrganizationID:    input.OrganizationID,
			SecretID:          input.SecretID,
			ClusterName:       input.ClusterName,
			ResourceGroupName: input.ResourceGroupName,
			RoleAssignment:    ra,
		}
		err := workflow.ExecuteActivity(ctx, AssignRoleActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	return nil
}
