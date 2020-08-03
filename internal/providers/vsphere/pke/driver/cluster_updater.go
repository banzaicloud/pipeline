// Copyright © 2020 Banzai Cloud
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
	"strings"
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence/client"

	"github.com/banzaicloud/pipeline/internal/providers/vsphere/pke"
	"github.com/banzaicloud/pipeline/internal/providers/vsphere/pke/driver/commoncluster"
	"github.com/banzaicloud/pipeline/internal/providers/vsphere/pke/workflow"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pipCluster "github.com/banzaicloud/pipeline/src/cluster"
	"github.com/banzaicloud/pipeline/src/secret"
	"github.com/banzaicloud/pipeline/src/utils"
)

type ClusterUpdater struct {
	logger         Logger
	paramsPreparer ClusterUpdateParamsPreparer
	config         ClusterConfig
	organizations  OrganizationStore
	secrets        ClusterUpdaterSecretStore
	store          pke.ClusterStore
	workflowClient client.Client
}

type ClusterUpdaterSecretStore interface {
	secretStore

	GetByName(organizationID uint, secretName string) (*secret.SecretItemResponse, error)
}

func MakeClusterUpdater(
	logger Logger,
	config ClusterConfig,
	organizations OrganizationStore,
	secrets ClusterUpdaterSecretStore,
	store pke.ClusterStore,
	workflowClient client.Client,
) ClusterUpdater {
	return ClusterUpdater{
		logger: logger,
		paramsPreparer: ClusterUpdateParamsPreparer{
			logger:  logger,
			secrets: secrets,
			store:   store,
		},
		config:         config,
		organizations:  organizations,
		secrets:        secrets,
		store:          store,
		workflowClient: workflowClient,
	}
}

type VspherePKEClusterUpdateParams struct {
	ClusterID uint
	NodePools []NodePool
}

func (cu ClusterUpdater) Update(ctx context.Context, params VspherePKEClusterUpdateParams) error {
	logger := cu.logger.WithFields(map[string]interface{}{"clusterID": params.ClusterID})

	logger.Info("updating cluster")

	if err := cu.paramsPreparer.Prepare(ctx, &params); err != nil {
		return errors.WrapIf(err, "params preparation failed")
	}

	cluster, err := cu.store.GetByID(params.ClusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to get cluster by ID")
	}

	nodePoolsToCreate, nodePoolsToUpdate, nodePoolsToDelete := sortNodePools(params.NodePools, cluster.NodePools)
	nodesToDelete := make([]workflow.Node, 0)
	var nodePoolLabels []pipCluster.NodePoolLabels

	sshKeyPair, err := GetOrCreateSSHKeyPair(cluster, cu.secrets, cu.store)
	if err != nil {
		return errors.WrapIf(err, "failed to get or create SSH key pair")
	}

	vsphereSecret, err := cu.secrets.Get(cluster.OrganizationID, cluster.SecretID)
	if err != nil {
		return errors.WrapIf(err, "failed to get cluster's secret")
	}
	defaultNodeTemplate := vsphereSecret.Values[secrettype.VsphereDefaultNodeTemplate]

	tf := nodeTemplateFactory{
		ClusterID:                   cluster.ID,
		ClusterName:                 cluster.Name,
		KubernetesVersion:           cluster.Kubernetes.Version,
		NoProxy:                     strings.Join(cluster.HTTPProxy.Exceptions, ","),
		OrganizationID:              cluster.OrganizationID,
		PipelineExternalURL:         cu.config.PipelineExternalURL,
		PipelineExternalURLInsecure: cu.config.PipelineExternalURLInsecure,
		SingleNodePool:              len(cluster.NodePools) == 1,
		SSHPublicKey:                sshKeyPair.PublicKeyData,
		LoadBalancerIPRange:         cluster.LoadBalancerIPRange,
	}

	nodesToCreate := make([]workflow.Node, 0)
	if len(nodePoolsToCreate) > 0 {
		if cluster.Kubernetes.OIDC.Enabled {
			tf.OIDCIssuerURL = cu.config.OIDCIssuerURL
			tf.OIDCClientID = cluster.UID
		}

		for _, np := range nodePoolsToCreate {
			nodePool := pke.NodePool{
				CreatedBy:    np.CreatedBy,
				Name:         np.Name,
				Roles:        np.Roles,
				Size:         np.Size,
				VCPU:         np.VCPU,
				RAM:          np.RAM,
				TemplateName: np.TemplateName,
			}
			if nodePool.TemplateName == "" {
				nodePool.TemplateName = defaultNodeTemplate
			}

			for i := 1; i <= np.Size; i++ {
				nodesToCreate = append(nodesToCreate, tf.getNode(nodePool, i))
			}

			nodePoolLabels = append(nodePoolLabels, pipCluster.NodePoolLabels{
				NodePoolName: np.Name,
				Existing:     false,
				InstanceType: np.InstanceType(),
				CustomLabels: np.Labels,
			})

			err := cu.store.CreateNodePool(cluster.ID, nodePool)
			if err != nil {
				err = errors.WrapIfWithDetails(err, "failed to create new node pool", "clusterID", cluster.ID, "nodePoolName", np.Name)
				_ = cu.handleError(cluster.ID, err)
			}
		}
	}

	// node pools to update
	if len(nodePoolsToUpdate) > 0 {
		existingNodePoolSet := make(map[string]pke.NodePool)
		for _, np := range cluster.NodePools {
			existingNodePoolSet[np.Name] = np
		}
		for _, np := range nodePoolsToUpdate {
			existingNodePool := existingNodePoolSet[np.Name]

			nodePoolLabels = append(nodePoolLabels, pipCluster.NodePoolLabels{
				NodePoolName: np.Name,
				Existing:     true,
				InstanceType: np.InstanceType(),
				CustomLabels: np.Labels,
			})

			if np.Size != existingNodePool.Size {
				// check existing nodes are fine, create new vm otherwise
				for i := 1; i <= np.Size; i++ {
					templateName := np.TemplateName
					if templateName == "" {
						templateName = defaultNodeTemplate
					}
					nodesToCreate = append(nodesToCreate, tf.getNode(pke.NodePool{
						CreatedBy:    np.CreatedBy,
						Name:         np.Name,
						Roles:        np.Roles,
						Size:         np.Size,
						VCPU:         np.VCPU,
						RAM:          np.RAM,
						TemplateName: templateName,
					}, i))
				}

				if existingNodePool.Size > np.Size {
					// delete unnecessary nodes
					for j := np.Size + 1; j <= existingNodePool.Size; j++ {
						nodesToDelete = append(nodesToDelete, workflow.Node{
							Name: pke.GetVMName(cluster.Name, np.Name, j),
						})
					}
				}

				err := cu.store.UpdateNodePoolSize(cluster.ID, np.Name, np.Size)
				if err != nil {
					err = errors.WrapIfWithDetails(err, "failed to update node pool size", "clusterID", cluster.ID, "nodePoolName", np.Name)
					_ = cu.handleError(cluster.ID, err)
				}
			}
		}
	}

	org, err := cu.organizations.Get(ctx, cluster.OrganizationID)
	if err != nil {
		return errors.WrapIf(err, "failed to get organization")
	}

	var labelsMap map[string]map[string]string
	{
		commonCluster, err := commoncluster.MakeCommonClusterGetter(cu.secrets, cu.store).GetByID(cluster.ID)
		if err != nil {
			return errors.WrapIf(err, "failed to get Azure PKE common cluster by ID")
		}

		labelsMap, err = pipCluster.GetDesiredLabelsForCluster(ctx, commonCluster, nodePoolLabels)
		if err != nil {
			return errors.WrapIf(err, "failed to get desired labels for cluster")
		}
	}

	input := workflow.UpdateClusterWorkflowInput{
		ClusterID:         cluster.ID,
		ClusterName:       cluster.Name,
		ClusterUID:        cluster.UID,
		OrganizationID:    cluster.OrganizationID,
		OrganizationName:  org.Name,
		SecretID:          cluster.SecretID,
		K8sSecretID:       cluster.K8sSecretID,
		StorageSecretID:   cluster.StorageSecretID,
		OIDCEnabled:       cluster.Kubernetes.OIDC.Enabled,
		MasterNodeNames:   getMasterNodeVMNames(cluster),
		NodesToCreate:     nodesToCreate,
		NodesToDelete:     nodesToDelete,
		NodePoolsToDelete: nodePoolsToDelete,
		HTTPProxy:         cluster.HTTPProxy,
		NodePoolLabels:    labelsMap,
		ResourcePoolName:  cluster.ResourcePool,
		DatastoreName:     cluster.Datastore,
		FolderName:        cluster.Folder,
	}

	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: 40 * time.Minute, // TODO: lower timeout
	}

	if err := cu.store.SetStatus(cluster.ID, pkgCluster.Updating, pkgCluster.UpdatingMessage); err != nil {
		return errors.WrapIf(err, "failed to set cluster status")
	}

	wfexec, err := cu.workflowClient.StartWorkflow(ctx, workflowOptions, workflow.UpdateClusterWorkflowName, input)
	if err := errors.WrapIfWithDetails(err, "failed to start workflow", "workflow", workflow.UpdateClusterWorkflowName); err != nil {
		_ = cu.handleError(cluster.ID, err)
		return err
	}

	if err := cu.store.SetActiveWorkflowID(cluster.ID, wfexec.ID); err != nil {
		err = errors.WrapIfWithDetails(err, "failed to set active workflow ID", "clusterID", cluster.ID, "workflowID", wfexec.ID)
		_ = cu.handleError(cluster.ID, err)
		return err
	}

	return nil
}

func (cu ClusterUpdater) handleError(clusterID uint, err error) error {
	return handleClusterError(cu.logger, cu.store, pkgCluster.Warning, clusterID, err)
}

func sortNodePools(incoming []NodePool, clusterNodePools []pke.NodePool) (toCreate, toUpdate []NodePool, toDelete []workflow.NodePool) {
	existingNodePoolSet := make(map[string]pke.NodePool)
	for _, np := range clusterNodePools {
		existingNodePoolSet[np.Name] = np
	}
	for _, np := range incoming {
		if _, ok := existingNodePoolSet[np.Name]; ok {
			delete(existingNodePoolSet, np.Name)
			toUpdate = append(toUpdate, np)
		} else {
			toCreate = append(toCreate, np)
		}
	}
	toDelete = make([]workflow.NodePool, 0, len(existingNodePoolSet))
	for _, np := range existingNodePoolSet {
		toDelete = append(toDelete, workflow.NodePool{
			Name: np.Name,
			Size: np.Size,
		})
	}
	return
}

type ClusterUpdateParamsPreparer struct {
	logger  Logger
	secrets interface {
		Get(organizationID uint, secretID string) (*secret.SecretItemResponse, error)
	}
	store pke.ClusterStore
}

func (p ClusterUpdateParamsPreparer) Prepare(ctx context.Context, params *VspherePKEClusterUpdateParams) error {
	if params.ClusterID == 0 {
		return validationErrorf("ClusterID cannot be 0")
	}
	cluster, err := p.store.GetByID(params.ClusterID)
	if pke.IsNotFound(err) {
		return validationErrorf("ClusterID must refer to an existing cluster")
	} else if err != nil {
		return errors.WrapIf(err, "failed to get cluster by ID")
	}

	nodePoolsPreparer := NodePoolsPreparer{
		logger: p.logger,
		dataProvider: clusterUpdaterNodePoolPreparerDataProvider{
			cluster: cluster,
		},
	}
	if err := nodePoolsPreparer.Prepare(ctx, params.NodePools); err != nil {
		return errors.WrapIf(err, "failed to prepare node pools")
	}
	return nil
}

type clusterUpdaterNodePoolPreparerDataProvider struct {
	cluster pke.PKEOnVsphereCluster
}

func (p clusterUpdaterNodePoolPreparerDataProvider) getExistingNodePools(ctx context.Context) ([]pke.NodePool, error) {
	return p.cluster.NodePools, nil
}

func (p clusterUpdaterNodePoolPreparerDataProvider) getExistingNodePoolByName(ctx context.Context, nodePoolName string) (pke.NodePool, error) {
	for _, np := range p.cluster.NodePools {
		if np.Name == nodePoolName {
			return np, nil
		}
	}
	return pke.NodePool{}, notExistsYetError{}
}

func getMasterNodeVMNames(cluster pke.PKEOnVsphereCluster) []string {
	masterVmNames := []string{}
	for _, np := range cluster.NodePools {
		names := []string{}
		for j := 1; j <= np.Size; j++ {
			names = append(names, pke.GetVMName(cluster.Name, np.Name, j))
		}

		if utils.Contains(np.Roles, "master") {
			masterVmNames = append(masterVmNames, names...)
		}
	}
	return masterVmNames
}
