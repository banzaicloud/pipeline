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
	"fmt"
	"net/http"
	"strings"
	"time"

	autoazure "github.com/Azure/go-autorest/autorest/azure"
	intPKE "github.com/banzaicloud/pipeline/internal/pke"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke/workflow"
	"github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/providers/azure"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
	"go.uber.org/cadence/client"
)

func NewAzurePKEClusterCreator(logger logrus.FieldLogger, store pke.AzurePKEClusterStore, workflowClient client.Client) AzurePKEClusterCreator {
	return AzurePKEClusterCreator{
		logger:         logger,
		paramsPreparer: MakeAzurePKEClusterCreationParamsPreparer(logger),
		store:          store,
		workflowClient: workflowClient,
	}
}

// AzurePKEClusterCreator creates new PKE-on-Azure clusters
type AzurePKEClusterCreator struct {
	logger         logrus.FieldLogger
	paramsPreparer AzurePKEClusterCreationParamsPreparer
	store          pke.AzurePKEClusterStore
	workflowClient client.Client
}

type VirtualNetwork struct {
	Name     string
	CIDR     string
	Location string
}

type NodePool struct {
	CreatedBy    uint
	Name         string
	InstanceType string
	Subnet       Subnet
	Zones        []string
	Roles        []string
	Labels       map[string]string
	Autoscaling  bool
	Count        int
	Min          int
	Max          int
}

type Subnet struct {
	Name string
	CIDR string
}

// AzurePKEClusterCreationParams defines parameters for PKE-on-Azure cluster creation
type AzurePKEClusterCreationParams struct {
	CreatedBy      uint
	Kubernetes     intPKE.Kubernetes
	Name           string
	Network        VirtualNetwork
	NodePools      []NodePool
	OrganizationID uint
	ResourceGroup  string
	ScaleOptions   cluster.ScaleOptions
	SecretID       string
	SSHSecretID    string
}

// Create
func (cc AzurePKEClusterCreator) Create(ctx context.Context, params AzurePKEClusterCreationParams) (cl pke.PKEOnAzureCluster, err error) {
	if err = cc.paramsPreparer.Prepare(ctx, &params); err != nil {
		return
	}
	createParams := pke.CreateParams{
		Name:           params.Name,
		OrganizationID: params.OrganizationID,
		CreatedBy:      params.CreatedBy,
		Location:       params.Network.Location,
		SecretID:       params.SecretID,
		SSHSecretID:    params.SSHSecretID,
		RBAC:           params.Kubernetes.RBAC,
		ScaleOptions:   params.ScaleOptions,
	}
	cl, err = cc.store.Create(createParams)
	if err != nil {
		return
	}

	input := workflow.CreateClusterWorkflowInput{
		OrganizationID: cl.OrganizationID,
		ClusterID:      cl.ID,
		ClusterUID:     cl.UID,
		ClusterName:    cl.Name,
		SecretID:       cl.SecretID,
		Location:       cl.Location,
	}

	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: 40 * time.Minute, // TODO: lower timeout
	}

	_, err = cc.workflowClient.ExecuteWorkflow(ctx, workflowOptions, workflow.CreateClusterWorkflowName, input)

	return
}

// AzurePKEClusterCreationParamsPreparer implements AzurePKEClusterCreationParams preparation
type AzurePKEClusterCreationParamsPreparer struct {
	k8sPreparer       intPKE.KubernetesPreparer
	logger            logrus.FieldLogger
	nodePoolsPreparer NodePoolsPreparer
}

// MakeAzurePKEClusterCreationParamsPreparer returns an instance of AzurePKEClusterCreationParamsPreparer
func MakeAzurePKEClusterCreationParamsPreparer(logger logrus.FieldLogger) AzurePKEClusterCreationParamsPreparer {
	return AzurePKEClusterCreationParamsPreparer{
		k8sPreparer: intPKE.MakeKubernetesPreparer(logger, "Kubernetes"),
		logger:      logger,
		nodePoolsPreparer: NodePoolsPreparer{
			logger:    logger,
			namespace: "NodePools",
		},
	}
}

func (p AzurePKEClusterCreationParamsPreparer) getVNetPreparer(cloudConnection *azure.CloudConnection, clusterName, resourceGroupName string) VirtualNetworkPreparer {
	return VirtualNetworkPreparer{
		clusterName:       clusterName,
		connection:        cloudConnection,
		logger:            p.logger,
		namespace:         "Network",
		resourceGroupName: resourceGroupName,
	}
}

// Prepare validates and provides defaults for AzurePKEClusterCreationParams fields
func (p AzurePKEClusterCreationParamsPreparer) Prepare(ctx context.Context, params *AzurePKEClusterCreationParams) error {
	if params.Name == "" {
		return validationErrorf("Name cannot be empty")
	}
	if params.OrganizationID == 0 {
		return validationErrorf("OrganizationID cannot be 0")
	}
	// TODO check org exists
	// TODO check creator user exists if present
	if params.SecretID == "" {
		return validationErrorf("SecretID cannot be empty")
	}
	// TODO validate secret ID
	// TODO validate SSH secret ID if present

	if params.ResourceGroup == "" {
		params.ResourceGroup = fmt.Sprintf("%s-rg", params.Name)
		p.logger.Debugf("ResourceGroup not specified, defaulting to [%s]", params.ResourceGroup)
	}

	if err := p.k8sPreparer.Prepare(&params.Kubernetes); err != nil {
		return emperror.Wrap(err, "failed to prepare k8s network")
	}

	sir, err := secret.Store.Get(params.OrganizationID, params.SecretID)
	if err != nil {
		return emperror.Wrap(err, "failed to fetch secret from store")
	}
	cc, err := azure.NewCloudConnection(&autoazure.PublicCloud, azure.NewCredentials(sir.Values))
	if err != nil {
		return emperror.Wrap(err, "failed to create Azure cloud connection")
	}
	if err := p.getVNetPreparer(cc, params.Name, params.ResourceGroup).Prepare(ctx, &params.Network); err != nil {
		return emperror.Wrap(err, "failed to prepare cluster network")
	}

	if err := p.nodePoolsPreparer.Prepare(params.NodePools); err != nil {
		return emperror.Wrap(err, "failed to prepare node pools")
	}

	return nil
}

// NodePoolsPreparer implements []NodePool preparation
type NodePoolsPreparer struct {
	logger    logrus.FieldLogger
	namespace string
}

func (p NodePoolsPreparer) getNodePoolPreparer(i int) NodePoolPreparer {
	return NodePoolPreparer{
		logger:    p.logger,
		namespace: fmt.Sprintf("%s[%d]", p.namespace, i),
	}
}

// Prepare validates and provides defaults for a set of NodePools
func (p NodePoolsPreparer) Prepare(nodePools []NodePool) error {
	for i := range nodePools {
		if err := p.getNodePoolPreparer(i).Prepare(&nodePools[i]); err != nil {
			return emperror.Wrap(err, "failed to prepare node pools")
		}
	}
	return nil
}

// NodePoolPreparer implements NodePool preparation
type NodePoolPreparer struct {
	logger    logrus.FieldLogger
	namespace string
}

// Prepare validates and provides defaults for NodePool fields
func (p NodePoolPreparer) Prepare(nodePool *NodePool) error {
	if nodePool == nil {
		return nil
	}

	for key, val := range nodePool.Labels {
		forbidden := ",:"
		if strings.ContainsAny(key, forbidden) {
			p.logger.Errorf("key [%s] in %s.Labels contains forbidden characters [%s]", key, p.namespace, forbidden)
			return validationErrorf("label [%s] contains forbidden characters [%s]", key, forbidden)
		}
		if strings.ContainsAny(val, forbidden) {
			p.logger.Errorf("value [%s] of %s.Labels[%q] contains forbidden characters [%s]", val, p.namespace, key, forbidden)
			return validationErrorf("value [%s] of label [%s] contains forbidden characters [%s]", val, key, forbidden)
		}
	}
	for i, r := range nodePool.Roles {
		forbidden := ","
		if strings.ContainsAny(r, forbidden) {
			p.logger.Errorf("value [%s] of %s.Roles[%d] contains forbidden characters [%s]", r, p.namespace, i, forbidden)
			return validationErrorf("role [%s] contains forbidden characters [%s]", r, forbidden)
		}
	}
	for i, z := range nodePool.Zones {
		forbidden := ","
		if strings.ContainsAny(z, forbidden) {
			p.logger.Errorf("value [%s] of %s.Zones[%d] contains forbidden characters [%s]", z, p.namespace, i, forbidden)
			return validationErrorf("zone [%s] contains forbidden characters [%s]", z, forbidden)
		}
	}
	return nil
}

// VirtualNetworkPreparer implements VirtualNetwork preparation
type VirtualNetworkPreparer struct {
	clusterName       string
	connection        *azure.CloudConnection
	logger            logrus.FieldLogger
	namespace         string
	resourceGroupName string
}

const DefaultVirtualNetworkCIDR = "10.0.0.0/16"

// Prepare validates and provides defaults for VirtualNetwork fields
func (p VirtualNetworkPreparer) Prepare(ctx context.Context, vnet *VirtualNetwork) error {
	if vnet.Name == "" {
		vnet.Name = fmt.Sprintf("%s-vnet", p.clusterName)
		p.logger.Debugf("%s.Name not specified, defaulting to [%s]", p.namespace, vnet.Name)
	}
	if vnet.CIDR == "" {
		vnet.CIDR = DefaultVirtualNetworkCIDR
		p.logger.Debugf("%s.CIDR not specified, defaulting to [%s]", p.namespace, vnet.CIDR)
	}
	if vnet.Location == "" {
		rg, err := p.connection.GetGroupsClient().Get(ctx, p.resourceGroupName)
		if err != nil && rg.Response.StatusCode != http.StatusNotFound {
			return emperror.WrapWith(err, "failed to fetch Azure resource group", "resourceGroupName", p.resourceGroupName)
		}
		if rg.Response.StatusCode == http.StatusNotFound || rg.Location == nil || *rg.Location == "" {
			// resource group does not exist (or somehow has no Location), cannot provide default
			return validationErrorf("%s.Location must be specified", p.namespace)
		}
		vnet.Location = *rg.Location
		p.logger.Debugf("%s.Location not specified, defaulting to resource group location [%s]", p.namespace, vnet.Location)
	}
	return nil
}

type validationError struct {
	msg string
}

func validationErrorf(msg string, args ...interface{}) validationError {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	return validationError{
		msg: msg,
	}
}

func (e validationError) Error() string {
	return e.msg
}

func (e validationError) InputValidationError() bool {
	return true
}
