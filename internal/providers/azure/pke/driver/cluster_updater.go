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

package driver

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke/workflow"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/providers/azure"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
	"go.uber.org/cadence/client"
)

type AzurePKEClusterUpdater struct {
	logger              logrus.FieldLogger
	paramsPreparer      AzurePKEClusterUpdateParamsPreparer
	pipelineExternalURL string
	secrets             clusterUpdaterSecretStore
	store               pke.AzurePKEClusterStore
	workflowClient      client.Client
}

type clusterUpdaterSecretStore interface {
	Get(organizationID uint, secretID string) (*secret.SecretItemResponse, error)
	Store(organizationID uint, request *secret.CreateSecretRequest) (string, error)
}

func MakeAzurePKEClusterUpdater(logger logrus.FieldLogger, pipelineExternalURL string, secrets clusterUpdaterSecretStore, store pke.AzurePKEClusterStore, workflowClient client.Client) AzurePKEClusterUpdater {
	return AzurePKEClusterUpdater{
		logger: logger,
		paramsPreparer: AzurePKEClusterUpdateParamsPreparer{
			logger: logger,
			store:  store,
		},
		pipelineExternalURL: pipelineExternalURL,
		secrets:             secrets,
		store:               store,
		workflowClient:      workflowClient,
	}
}

type AzurePKEClusterUpdateParams struct {
	ClusterID uint
	NodePools []NodePool
}

func (cu AzurePKEClusterUpdater) Update(ctx context.Context, params AzurePKEClusterUpdateParams) error {
	logger := cu.logger.WithField("clusterID", params.ClusterID)

	logger.Info("updating cluster")

	if err := cu.paramsPreparer.Prepare(ctx, &params); err != nil {
		return emperror.Wrap(err, "params preparation failed")
	}

	cluster, err := cu.store.GetByID(params.ClusterID)
	if err != nil {
		return emperror.Wrap(err, "failed to get cluster by ID")
	}

	nodePoolsToCreate, nodePoolsToUpdate, nodePoolsToDelete := sortNodePools(params.NodePools, cluster.NodePools)
	subnetsToCreate, subnetsToDelete := sortSubnets(nodePoolsToCreate, nodePoolsToUpdate, nodePoolsToDelete)

	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: 40 * time.Minute, // TODO: lower timeout
	}

	sshKeyPair, err := GetOrCreateSSHKeyPair(cluster, cu.secrets, cu.store)
	if err != nil {
		return emperror.Wrap(err, "failed to get or create SSH key pair")
	}

	sir, err := cu.secrets.Get(cluster.OrganizationID, cluster.SecretID)
	if err != nil {
		return emperror.Wrap(err, "failed to get cluster secret")
	}
	tenantID := sir.GetValue(pkgSecret.AzureTenantID)

	toCreateVMSSTemplates := make([]workflow.VirtualMachineScaleSetTemplate, len(nodePoolsToCreate))
	var toCreateSubnetTemplates []workflow.SubnetTemplate
	var roleAssignmentTemplates []workflow.RoleAssignmentTemplate
	{
		subnetTemplates := make(map[string]workflow.SubnetTemplate)

		tf := nodePoolTemplateFactory{
			ClusterID:           cluster.ID,
			ClusterName:         cluster.Name,
			KubernetesVersion:   cluster.Kubernetes.Version,
			Location:            cluster.Location,
			OrganizationID:      cluster.OrganizationID,
			PipelineExternalURL: cu.pipelineExternalURL,
			ResourceGroupName:   cluster.ResourceGroup.Name,
			SingleNodePool:      (len(nodePoolsToCreate) + len(nodePoolsToUpdate) - len(nodePoolsToDelete)) == 1,
			SSHPublicKey:        sshKeyPair.PublicKeyData,
			TenantID:            tenantID,
			VirtualNetworkName:  cluster.VirtualNetwork.Name,
		}

		for i, np := range nodePoolsToCreate {
			vmsst, snt, rats := tf.getTemplates(np)
			toCreateVMSSTemplates[i] = vmsst
			if subnetsToCreate[snt.Name] {
				subnetTemplates[snt.Name] = snt
			}
			roleAssignmentTemplates = append(roleAssignmentTemplates, rats...)

			err := cu.store.CreateNodePool(params.ClusterID, np.toPke())
			if err != nil {
				return emperror.WrapWith(err, "failed to store new node pool", "clusterID", cluster.ID, "nodepool", np.Name)
			}
		}

		toCreateSubnetTemplates = make([]workflow.SubnetTemplate, 0, len(subnetTemplates))
		for _, t := range subnetTemplates {
			toCreateSubnetTemplates = append(toCreateSubnetTemplates, t)
		}
	}

	toUpdateVMSSChanges := make([]workflow.VirtualMachineScaleSetChanges, len(nodePoolsToUpdate))
	for i, np := range nodePoolsToUpdate {
		toUpdateVMSSChanges[i] = workflow.VirtualMachineScaleSetChanges{
			Name:          pke.GetVMSSName(cluster.Name, np.Name),
			InstanceCount: uint(np.Count),
		}

		err := cu.store.SetNodePoolSizes(params.ClusterID, np.Name, uint(np.Min), uint(np.Max), uint(np.Count), np.Autoscaling)
		if err != nil {
			return emperror.WrapWith(err, "failed to store updated node pool", "clusterID", cluster.ID, "nodepool", np.Name)
		}
	}

	toDeleteVMSSNames := make([]string, len(nodePoolsToDelete))
	for i, np := range nodePoolsToDelete {
		toDeleteVMSSNames[i] = np.Name
		// will only be persisted by the successful workflow
	}

	input := workflow.UpdateClusterWorkflowInput{
		OrganizationID:     cluster.OrganizationID,
		SecretID:           cluster.SecretID,
		ClusterID:          cluster.ID,
		ClusterName:        cluster.Name,
		ResourceGroupName:  cluster.ResourceGroup.Name,
		VirtualNetworkName: cluster.VirtualNetwork.Name,

		RoleAssignments: roleAssignmentTemplates,
		SubnetsToCreate: toCreateSubnetTemplates,
		SubnetsToDelete: subnetsToDelete,
		VMSSToCreate:    toCreateVMSSTemplates,
		VMSSToDelete:    toDeleteVMSSNames,
		VMSSToUpdate:    toUpdateVMSSChanges,
	}

	if err := cu.store.SetStatus(cluster.ID, pkgCluster.Updating, pkgCluster.UpdatingMessage); err != nil {
		return emperror.Wrap(err, "failed to set cluster status")
	}

	wfexec, err := cu.workflowClient.StartWorkflow(ctx, workflowOptions, workflow.UpdateClusterWorkflowName, input)
	if err := emperror.WrapWith(err, "failed to start workflow", "workflow", workflow.UpdateClusterWorkflowName); err != nil {
		cu.handleError(cluster.ID, err)
		return err
	}

	if err := cu.store.SetActiveWorkflowID(cluster.ID, wfexec.ID); err != nil {
		err = emperror.WrapWith(err, "failed to set active workflow ID", "clusterID", cluster.ID, "workflowID", wfexec.ID)
		cu.handleError(cluster.ID, err)
		return err
	}

	return nil
}

func (cu AzurePKEClusterUpdater) handleError(clusterID uint, err error) error {
	return handleClusterError(cu.logger, cu.store, pkgCluster.Warning, clusterID, err)
}

func sortNodePools(incoming []NodePool, existing []pke.NodePool) (toCreate, toUpdate []NodePool, toDelete []pke.NodePool) {
	existingSet := make(map[string]pke.NodePool)
	for _, np := range existing {
		existingSet[np.Name] = np
	}
	for _, np := range incoming {
		if _, ok := existingSet[np.Name]; ok {
			delete(existingSet, np.Name)
			toUpdate = append(toUpdate, np)
		} else {
			toCreate = append(toCreate, np)
		}
	}
	toDelete = make([]pke.NodePool, 0, len(existingSet))
	for _, np := range existingSet {
		toDelete = append(toDelete, np)
	}
	return
}

func sortSubnets(nodePoolsToCreate, nodePoolsToUpdate []NodePool, nodePoolsToDelete []pke.NodePool) (toCreate map[string]bool, toDelete []string) {
	// sentence to-be-deleted node pools' subnets to deletion
	toDeleteSet := make(map[string]bool)
	for _, np := range nodePoolsToDelete {
		toDeleteSet[np.Subnet.Name] = true
	}

	// add to-be-updated node pools' subnets to the set of subnets we keep
	// additionally, if the subnet was to be deleted, keep it from deletion
	toKeep := make(map[string]bool)
	for _, np := range nodePoolsToUpdate {
		if toDeleteSet[np.Subnet.Name] {
			delete(toDeleteSet, np.Subnet.Name)
		}
		toKeep[np.Subnet.Name] = true
	}

	// if a to-be-created node pool referes to a to-be-deleted subnet, keep the subnet from deletion
	// otherwise, if the subnet is not in the to-be-kept set, it must be created
	toCreate = make(map[string]bool)
	for _, np := range nodePoolsToCreate {
		if toDeleteSet[np.Subnet.Name] {
			delete(toDeleteSet, np.Subnet.Name)
		} else if !toKeep[np.Subnet.Name] {
			toCreate[np.Subnet.Name] = true
		}
	}

	toDelete = make([]string, 0, len(toDeleteSet))
	for name := range toDeleteSet {
		toDelete = append(toDelete, name)
	}

	return
}

type AzurePKEClusterUpdateParamsPreparer struct {
	logger logrus.FieldLogger
	store  pke.AzurePKEClusterStore
}

func (p AzurePKEClusterUpdateParamsPreparer) Prepare(ctx context.Context, params *AzurePKEClusterUpdateParams) error {
	if params.ClusterID == 0 {
		return validationErrorf("ClusterID cannot be 0")
	}
	cluster, err := p.store.GetByID(params.ClusterID)
	if pke.IsNotFound(err) {
		return validationErrorf("ClusterID must refer to an existing cluster")
	} else if err != nil {
		return emperror.Wrap(err, "failed to get cluster by ID")
	}
	nodePoolsPreparer := NodePoolsPreparer{
		logger:    p.logger,
		namespace: "NodePools",
		dataProvider: clusterUpdaterNodePoolPreparerDataProvider{
			cluster: cluster,
		},
	}
	if err := nodePoolsPreparer.Prepare(ctx, params.NodePools); err != nil {
		return emperror.Wrap(err, "failed to prepare node pools")
	}
	return nil
}

type clusterUpdaterNodePoolPreparerDataProvider struct {
	cluster               pke.PKEOnAzureCluster
	resourceGroupName     string
	subnetsClient         azure.SubnetsClient
	virtualNetworkName    string
	virtualNetworksClient azure.VirtualNetworksClient
}

func (p clusterUpdaterNodePoolPreparerDataProvider) getExistingNodePoolByName(ctx context.Context, nodePoolName string) (pke.NodePool, error) {
	for _, np := range p.cluster.NodePools {
		if np.Name == nodePoolName {
			return np, nil
		}
	}
	return pke.NodePool{}, notExistsYetError{}
}

func (p clusterUpdaterNodePoolPreparerDataProvider) getSubnetCIDR(ctx context.Context, nodePool pke.NodePool) (string, error) {
	subnet, err := p.subnetsClient.Get(ctx, p.resourceGroupName, p.virtualNetworkName, nodePool.Subnet.Name, "")
	if err != nil {
		return "", emperror.Wrap(err, "failed to get subnet")
	}
	return to.String(subnet.AddressPrefix), nil
}

func (p clusterUpdaterNodePoolPreparerDataProvider) getVirtualNetworkAddressRange(ctx context.Context) (net.IPNet, error) {
	vnet, err := p.virtualNetworksClient.Get(ctx, p.resourceGroupName, p.virtualNetworkName, "")
	if err != nil {
		return net.IPNet{}, emperror.Wrap(err, "failed to get virtual network")
	}
	if f := vnet.VirtualNetworkPropertiesFormat; f != nil {
		if as := f.AddressSpace; as != nil {
			if apsp := as.AddressPrefixes; apsp != nil {
				aps := to.StringSlice(apsp)
				if len(aps) > 0 {
					_, n, err := net.ParseCIDR(aps[0])
					if err != nil {
						return net.IPNet{}, emperror.Wrap(err, "failed to parse CIDR")
					}
					return *n, nil
				}
			}
		}
	}
	return net.IPNet{}, emperror.With(errors.New("virtual network has no address prefixes"), "resourceGroup", p.resourceGroupName, "vnet", p.virtualNetworkName)
}
