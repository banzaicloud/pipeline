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
	"strings"
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/clustersetup"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	pkgCadence "github.com/banzaicloud/pipeline/pkg/cadence"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/sdk/brn"
	"github.com/banzaicloud/pipeline/src/cluster"
)

const UpdateClusterWorkflowName = "pke-azure-update-cluster"

// UpdateClusterWorkflowInput
type UpdateClusterWorkflowInput struct {
	OrganizationID      uint
	SecretID            string
	ClusterID           uint
	ClusterName         string
	ConfigSecretID      string
	ResourceGroupName   string
	PublicIPAddressName string
	RouteTableName      string
	VirtualNetworkName  string

	RoleAssignments []RoleAssignmentTemplate
	SubnetsToCreate []SubnetTemplate
	SubnetsToDelete []string
	VMSSToCreate    []VirtualMachineScaleSetTemplate
	VMSSToDelete    []NodePoolAndVMSS
	VMSSToUpdate    []VirtualMachineScaleSetChanges

	Labels map[string]map[string]string

	AccessPoints          pke.AccessPoints
	APIServerAccessPoints pke.APIServerAccessPoints
}

type NodePoolAndVMSS struct {
	NodePoolName string
	VMSSName     string
}

func UpdateClusterWorkflow(ctx workflow.Context, input UpdateClusterWorkflowInput) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		ScheduleToCloseTimeout: 15 * time.Minute,
		WaitForCancellation:    true,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var providers CollectUpdateClusterProvidersActivityOutput
	{
		activityInput := CollectUpdateClusterProvidersActivityInput{
			OrganizationID:      input.OrganizationID,
			SecretID:            input.SecretID,
			ResourceGroupName:   input.ResourceGroupName,
			ClusterName:         input.ClusterName,
			PublicIPAddressName: input.PublicIPAddressName,
			RouteTableName:      input.RouteTableName,
			VirtualNetworkName:  input.VirtualNetworkName,
		}
		err := workflow.ExecuteActivity(ctx, CollectUpdateClusterProvidersActivityName, activityInput).Get(ctx, &providers)
		if err != nil {
			err = errors.WrapIff(pkgCadence.UnwrapError(err), "%q activity failed", CollectUpdateClusterProvidersActivityName)
			setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error()) // nolint: errcheck
			return err
		}
	}
	{
		futures := make([]workflow.Future, len(input.VMSSToDelete))

		for i, npAndVMSS := range input.VMSSToDelete {
			activityInput := DeleteVMSSActivityInput{
				OrganizationID:    input.OrganizationID,
				SecretID:          input.SecretID,
				ClusterName:       input.ClusterName,
				ResourceGroupName: input.ResourceGroupName,
				VMSSName:          npAndVMSS.VMSSName,
			}
			futures[i] = workflow.ExecuteActivity(ctx, DeleteVMSSActivityName, activityInput)
		}

		errs := make([]error, len(futures))

		for i, f := range futures {
			errs[i] = errors.WrapIff(pkgCadence.UnwrapError(f.Get(ctx, nil)), "%q activity failed", DeleteVMSSActivityName)
		}

		if err := errors.Combine(errs...); err != nil {
			setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error()) // nolint: errcheck
			return err
		}
	}
	{
		nodePoolsToDelete := make([]string, len(input.VMSSToDelete))
		for i, npAndVMSS := range input.VMSSToDelete {
			nodePoolsToDelete[i] = npAndVMSS.NodePoolName
		}
		activityInput := DeleteNodePoolFromStoreActivityInput{
			ClusterID:     input.ClusterID,
			NodePoolNames: nodePoolsToDelete,
		}
		if err := workflow.ExecuteActivity(ctx, DeleteNodePoolFromStoreActivityName, activityInput).Get(ctx, nil); err != nil {
			err = errors.WrapIff(pkgCadence.UnwrapError(err), "%q activity failed", DeleteNodePoolFromStoreActivityName)
			setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error()) // nolint: errcheck
			return err
		}
	}
	{
		futures := make([]workflow.Future, len(input.SubnetsToDelete))

		for i, subnetName := range input.SubnetsToDelete {
			activityInput := DeleteSubnetActivityInput{
				OrganizationID:    input.OrganizationID,
				SecretID:          input.SecretID,
				ClusterName:       input.ClusterName,
				ResourceGroupName: input.ResourceGroupName,
				VNetName:          input.VirtualNetworkName,
				SubnetName:        subnetName,
			}
			futures[i] = workflow.ExecuteActivity(ctx, DeleteSubnetActivityName, activityInput)
		}

		errs := make([]error, len(futures))

		for i, f := range futures {
			errs[i] = errors.WrapIff(pkgCadence.UnwrapError(f.Get(ctx, nil)), "%q activity failed", DeleteSubnetActivityName)
		}

		if err := errors.Combine(errs...); err != nil {
			setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error()) // nolint: errcheck
			return err
		}
	}

	// set up node pool labels set
	{
		activityInput := clustersetup.ConfigureNodePoolLabelsActivityInput{
			ConfigSecretID: brn.New(input.OrganizationID, brn.SecretResourceType, input.ConfigSecretID).String(),
			Labels:         input.Labels,
		}
		err := workflow.ExecuteActivity(ctx, clustersetup.ConfigureNodePoolLabelsActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			err = errors.WrapIff(pkgCadence.UnwrapError(err), "%q activity failed", clustersetup.ConfigureNodePoolLabelsActivityName)
			setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error()) // nolint: errcheck
			return err
		}
	}

	{
		futures := make([]workflow.Future, len(input.VMSSToUpdate))

		for i, vmssChanges := range input.VMSSToUpdate {
			activityInput := UpdateVMSSActivityInput{
				OrganizationID:    input.OrganizationID,
				SecretID:          input.SecretID,
				ClusterName:       input.ClusterName,
				ResourceGroupName: input.ResourceGroupName,
				Changes:           vmssChanges,
			}
			futures[i] = workflow.ExecuteActivity(ctx, UpdateVMSSActivityName, activityInput)
		}

		errs := make([]error, len(futures))

		for i, f := range futures {
			errs[i] = errors.WrapIff(pkgCadence.UnwrapError(f.Get(ctx, nil)), "%q activity failed", UpdateVMSSActivityName)
		}

		if err := errors.Combine(errs...); err != nil {
			setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error()) // nolint: errcheck
			return err
		}
	}

	{
		futures := make([]workflow.Future, len(input.SubnetsToCreate))

		for i, subnet := range input.SubnetsToCreate {
			activityInput := CreateSubnetActivityInput{
				OrganizationID:     input.OrganizationID,
				SecretID:           input.SecretID,
				ClusterName:        input.ClusterName,
				ResourceGroupName:  input.ResourceGroupName,
				VirtualNetworkName: input.VirtualNetworkName,
				Subnet:             subnet.Render(providers.RouteTableIDProvider, providers.SecurityGroupIDProvider),
			}

			futures[i] = workflow.ExecuteActivity(ctx, CreateSubnetActivityName, activityInput)
		}

		errs := make([]error, len(futures))

		for i, future := range futures {
			var activityOutput CreateSubnetActivityOutput

			errs[i] = errors.WrapIff(pkgCadence.UnwrapError(future.Get(ctx, &activityOutput)), "%q activity failed", CreateSubnetActivityName)

			providers.SubnetIDProvider.Put(input.SubnetsToCreate[i].Name, activityOutput.SubnetID)
		}

		if err := errors.Combine(errs...); err != nil {
			setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error()) // nolint: errcheck
			return err
		}
	}

	createdVMSSOutputs := make(map[string]CreateVMSSActivityOutput)
	{
		var apiServerPublicAddressProvider, apiServerPrivateAddressProvider IPAddressProvider
		apiServerCertSansMap := make(map[string]bool)

		if input.APIServerAccessPoints.Exists("public") && input.AccessPoints.Get("public").Address != "" {
			apiServerPublicAddressProvider = ConstantIPAddressProvider(input.AccessPoints.Get("public").Address)
			apiServerCertSansMap[input.AccessPoints.Get("public").Address] = true
		}

		if input.APIServerAccessPoints.Exists("private") && input.AccessPoints.Get("private").Address != "" {
			apiServerPrivateAddressProvider = ConstantIPAddressProvider(input.AccessPoints.Get("private").Address)
			apiServerCertSansMap[input.AccessPoints.Get("private").Address] = true
		}

		var apiServerCertSans []string
		for certSan := range apiServerCertSansMap {
			apiServerCertSans = append(apiServerCertSans, certSan)
		}
		apiServerCertSansProvider := ConstantResourceIDProvider(strings.Join(apiServerCertSans, ","))

		futures := make([]workflow.Future, len(input.VMSSToCreate))

		for i, vmss := range input.VMSSToCreate {
			var apiServerAddressProvider IPAddressProvider
			if apiServerPrivateAddressProvider != nil {
				apiServerAddressProvider = apiServerPrivateAddressProvider
			} else if apiServerPublicAddressProvider != nil {
				apiServerAddressProvider = apiServerPublicAddressProvider
			} else {
				return errors.New("no API server address available")
			}

			backendAddressPoolIDProviders := make([]ResourceIDByNameProvider, len(providers.BackendAddressPoolIDProviders))
			for i := range providers.BackendAddressPoolIDProviders {
				backendAddressPoolIDProviders[i] = providers.BackendAddressPoolIDProviders[i]
			}
			inboundNATPoolIDProviders := make([]ResourceIDByNameProvider, len(providers.InboundNATPoolIDProviders))
			for i := range providers.InboundNATPoolIDProviders {
				inboundNATPoolIDProviders[i] = providers.InboundNATPoolIDProviders[i]
			}

			activityInput := CreateVMSSActivityInput{
				OrganizationID:    input.OrganizationID,
				SecretID:          input.SecretID,
				ClusterID:         input.ClusterID,
				ClusterName:       input.ClusterName,
				ResourceGroupName: input.ResourceGroupName,
				ScaleSet:          vmss.Render(backendAddressPoolIDProviders, inboundNATPoolIDProviders, apiServerAddressProvider, apiServerCertSansProvider, providers.SecurityGroupIDProvider, providers.SubnetIDProvider),
			}

			futures[i] = workflow.ExecuteActivity(ctx, CreateVMSSActivityName, activityInput)
		}

		errs := make([]error, len(futures))
		var nodePoolsToDelete []string

		for i, future := range futures {
			var activityOutput CreateVMSSActivityOutput

			errs[i] = errors.WrapIff(pkgCadence.UnwrapError(future.Get(ctx, &activityOutput)), "%q activity failed", CreateVMSSActivityName)

			if errs[i] != nil {
				nodePoolsToDelete = append(nodePoolsToDelete, input.VMSSToCreate[i].NodePoolName)
			} else {
				createdVMSSOutputs[input.VMSSToCreate[i].Name] = activityOutput
			}
		}

		if err := errors.Combine(errs...); err != nil {
			setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error()) // nolint: errcheck

			{
				activityInput := DeleteNodePoolFromStoreActivityInput{
					ClusterID:     input.ClusterID,
					NodePoolNames: nodePoolsToDelete,
				}

				if err := workflow.ExecuteActivity(ctx, DeleteNodePoolFromStoreActivityName, activityInput).Get(ctx, nil); err != nil {
					err = errors.WrapIff(pkgCadence.UnwrapError(err), "%q activity failed", DeleteNodePoolFromStoreActivityName)

					setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error()) // nolint: errcheck
				}
			}

			return err
		}
	}

	{
		futures := make([]workflow.Future, len(input.RoleAssignments))

		for i, ra := range input.RoleAssignments {
			activityInput := AssignRoleActivityInput{
				OrganizationID:    input.OrganizationID,
				SecretID:          input.SecretID,
				ClusterName:       input.ClusterName,
				ResourceGroupName: input.ResourceGroupName,
				RoleAssignment:    ra.Render(mapVMSSPrincipalIDProvider(createdVMSSOutputs)),
			}
			futures[i] = workflow.ExecuteActivity(ctx, AssignRoleActivityName, activityInput)
		}

		errs := make([]error, len(futures))

		for i, f := range futures {
			errs[i] = errors.WrapIff(pkgCadence.UnwrapError(f.Get(ctx, nil)), "%q activity failed", AssignRoleActivityName)
		}

		if err := errors.Combine(errs...); err != nil {
			setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error()) // nolint: errcheck
			return err
		}
	}
	// redeploy autoscaler
	{
		activityInput := cluster.RunPostHookActivityInput{
			ClusterID: input.ClusterID,
			HookName:  pkgCluster.InstallClusterAutoscalerPostHook,
			Status:    pkgCluster.Updating,
		}

		err := workflow.ExecuteActivity(ctx, cluster.RunPostHookActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			err = errors.WrapIff(pkgCadence.UnwrapError(err), "%q activity failed", cluster.RunPostHookActivityName)
			setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error()) // nolint: errcheck
			return err
		}
	}

	setClusterStatus(ctx, input.ClusterID, pkgCluster.Running, pkgCluster.RunningMessage) // nolint: errcheck

	return nil
}
