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
	"github.com/gofrs/uuid"
	"net/http"
	"strconv"
	"strings"
	"time"

	autoazure "github.com/Azure/go-autorest/autorest/azure"
	intPKE "github.com/banzaicloud/pipeline/internal/pke"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke/workflow"
	"github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/providers/azure"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
	"go.uber.org/cadence/client"
)

func NewAzurePKEClusterCreator(logger logrus.FieldLogger, store pke.AzurePKEClusterStore, workflowClient client.Client, pipelineExternalURL string) AzurePKEClusterCreator {
	return AzurePKEClusterCreator{
		logger:              logger,
		paramsPreparer:      MakeAzurePKEClusterCreationParamsPreparer(logger),
		store:               store,
		workflowClient:      workflowClient,
		pipelineExternalURL: pipelineExternalURL,
	}
}

// AzurePKEClusterCreator creates new PKE-on-Azure clusters
type AzurePKEClusterCreator struct {
	logger              logrus.FieldLogger
	paramsPreparer      AzurePKEClusterCreationParamsPreparer
	store               pke.AzurePKEClusterStore
	workflowClient      client.Client
	pipelineExternalURL string
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
		Name:              params.Name,
		OrganizationID:    params.OrganizationID,
		CreatedBy:         params.CreatedBy,
		Location:          params.Network.Location,
		SecretID:          params.SecretID,
		SSHSecretID:       params.SSHSecretID,
		RBAC:              params.Kubernetes.RBAC,
		ScaleOptions:      params.ScaleOptions,
		ResourceGroupName: params.ResourceGroup,
	}
	cl, err = cc.store.Create(createParams)
	if err != nil {
		return
	}

	var sshKeyPair secret.SSHKeyPair
	if params.SSHSecretID == "" {
		sshKeyPair, sshSecretID, err := newSSHKeyPair(cl.OrganizationID, cl.ID, cl.Name, cl.UID)
		if err != nil {
			return
		}
		if err = cc.store.SetSSHSecretID(cl.ID, sshSecretID); err != nil {
			return
		}
	} else {
		sshKeyPair, err = getSSHKeyPair(cl.OrganizationID, params.SSHSecretID)
		if err != nil {
			return
		}
	}
	sshPublicKey := sshKeyPair.PublicKeyData

	sir, err := secret.Store.Get(params.OrganizationID, params.SecretID)
	if err != nil {
		return
	}
	tenantID := sir.GetValue(pkgSecret.AzureTenantID)

	input := workflow.CreateClusterWorkflowInput{
		ClusterID:         cl.ID,
		ClusterName:       params.Name,
		OrganizationID:    params.OrganizationID,
		ResourceGroupName: params.ResourceGroup,
		SecretID:          params.SecretID,
		VirtualNetworkTemplate: workflow.VirtualNetworkTemplate{
			Name: params.Name + "-vnet",
			CIDRs: []string{
				"10.240.0.0/16",
			},
			Location: params.Network.Location,
			Subnets: []workflow.SubnetTemplate{
				{
					Name:                     "master-subnet",
					CIDR:                     "10.240.0.0/24",
					NetworkSecurityGroupName: params.Name + "master-nsg",
				},
				{
					Name:                     "worker-subnet",
					CIDR:                     "10.240.1.0/24",
					NetworkSecurityGroupName: params.Name + "worker-nsg",
				},
			},
		},
		LoadBalancerTemplate: workflow.LoadBalancerTemplate{
			Name:                   params.Name + "-lb",
			Location:               params.Network.Location,
			SKU:                    "Standard",
			BackendAddressPoolName: "backend-address-pool",
			InboundNATPoolName:     "ssh-inbound-nat-pool",
		},
		PublicIPAddress: workflow.PublicIPAddress{
			Location: params.Network.Location,
			Name:     params.Name + "-pip-in",
			SKU:      "Standard",
		},
		RoleAssignmentTemplates: []workflow.RoleAssignmentTemplate{
			{
				Name:     uuid.Must(uuid.NewV1()).String(),
				VMSSName: params.Name + "master-vmss",
				RoleName: "Contributor",
			},
			{
				Name:     uuid.Must(uuid.NewV1()).String(),
				VMSSName: params.Name + "worker-vmss",
				RoleName: "Contributor",
			},
		},
		RouteTable: workflow.RouteTable{
			Name:     params.Name + "-route-table",
			Location: params.Network.Location,
		},
		SecurityGroups: []workflow.SecurityGroup{
			{
				Name:     params.Name + "master-nsg",
				Location: params.Network.Location,
				Rules: []workflow.SecurityRule{
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
			{
				Name:     params.Name + "worker-nsg",
				Location: params.Network.Location,
				Rules:    []workflow.SecurityRule{},
			},
		},
		VirtualMachineScaleSetTemplates: []workflow.VirtualMachineScaleSetTemplate{
			{
				AdminUsername: "azureuser",
				Image: workflow.Image{
					Offer:     "CentOS-CI",
					Publisher: "OpenLogic",
					SKU:       "7-CI",
					Version:   "7.6.20190306",
				},
				InstanceCount:            1,
				InstanceType:             "Standard_B2s",
				BackendAddressPoolName:   "backend-address-pool",
				InboundNATPoolName:       "ssh-inbound-nat-pool",
				Location:                 params.Network.Location,
				Name:                     params.Name + "master-vmss",
				NetworkSecurityGroupName: params.Name + "-master-nsg",
				SSHPublicKey:             sshPublicKey,
				SubnetName:               "master-subnet",
				UserDataScriptParams: map[string]string{
					"ClusterID":             strconv.FormatUint(uint64(cl.ID), 10),
					"InfraCIDR":             "10.240.0.0/24",
					"LoadBalancerSKU":       "standard",
					"NodePoolName":          "master-node-pool",
					"NSGName":               params.Name + "-worker-nsg",
					"OrgID":                 strconv.FormatUint(uint64(params.OrganizationID), 10),
					"PipelineURL":           cc.pipelineExternalURL,
					"PipelineToken":         "<not yet set>",
					"PKEVersion":            "0.4.0", // TODO: remove hard-coded constant
					"PublicAddress":         "<not yet set>",
					"RouteTableName":        params.Name + "-route-table",
					"SubnetName":            "master-subnet",
					"TenantID":              tenantID,
					"VnetName":              params.Name + "-vnet",
					"VnetResourceGroupName": params.ResourceGroup,
				},
				UserDataScriptTemplate: masterUserDataScriptTemplate,
				Zones:                  []string{"1", "2", "3"},
			},
			{
				AdminUsername: "azureuser",
				Image: workflow.Image{
					Offer:     "CentOS-CI",
					Publisher: "OpenLogic",
					SKU:       "7-CI",
					Version:   "7.6.20190306",
				},
				InstanceCount:            1,
				InstanceType:             "Standard_B2s",
				Location:                 params.Network.Location,
				Name:                     params.Name + "worker-vmss",
				NetworkSecurityGroupName: params.Name + "-worker-nsg",
				SSHPublicKey:             sshPublicKey,
				SubnetName:               "worker-subnet",
				UserDataScriptParams: map[string]string{
					"ClusterID":             strconv.FormatUint(uint64(cl.ID), 10),
					"InfraCIDR":             "10.240.1.0/24",
					"LoadBalancerSKU":       "standard",
					"NodePoolName":          "worker-node-pool",
					"NSGName":               params.Name + "-worker-nsg",
					"OrgID":                 strconv.FormatUint(uint64(params.OrganizationID), 10),
					"PipelineURL":           cc.pipelineExternalURL,
					"PipelineToken":         "<not yet set>",
					"PKEVersion":            "0.4.0", // TODO: remove hard-coded constant
					"PublicAddress":         "<not yet set>",
					"RouteTableName":        params.Name + "-route-table",
					"SubnetName":            "worker-subnet",
					"TenantID":              tenantID,
					"VnetName":              params.Name + "-vnet",
					"VnetResourceGroupName": params.ResourceGroup,
				},
				UserDataScriptTemplate: workerUserDataScriptTemplate,
				Zones:                  []string{"1", "2", "3"},
			},
		},
	}
	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: 40 * time.Minute, // TODO: lower timeout
	}

	wfexec, err := cc.workflowClient.StartWorkflow(ctx, workflowOptions, workflow.CreateClusterWorkflowName, input)
	if err != nil {
		return
	}

	err = cc.store.SetActiveWorkflowID(cl.ID, wfexec.ID)
	if err != nil {
		return
	}

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

const masterUserDataScriptTemplate = `#!/bin/sh
curl -v https://banzaicloud.com/downloads/pke/pke-{{ .PKEVersion }} -o /usr/local/bin/pke
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

const workerUserDataScriptTemplate = `
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
--kubernetes-pod-network-cidr=""`

func getSSHKeyPair(orgID uint, sshSecretID string) (*secret.SSHKeyPair, error) {
	sir, err := secret.Store.Get(orgID, sshSecretID)
	if err != nil {
		return nil, err
	}
	return secret.NewSSHKeyPair(sir), nil
}

func newSSHKeyPair(orgID uint, clusterID uint, clusterName string, clusterUID string) (sshKeyPair *secret.SSHKeyPair, sshSecretID string, err error) {
	sshKeyPair, err = secret.GenerateSSHKeyPair()
	if err != nil {
		return
	}
	sshSecretID, err = secret.StoreSSHKeyPair(sshKeyPair, orgID, clusterID, clusterName, clusterUID)
	return
}
