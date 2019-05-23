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
	"net"
	"time"

	autoazure "github.com/Azure/go-autorest/autorest/azure"
	"github.com/banzaicloud/pipeline/cluster"
	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	intPKE "github.com/banzaicloud/pipeline/internal/pke"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke/driver/commoncluster"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke/workflow"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgPKE "github.com/banzaicloud/pipeline/pkg/cluster/pke"
	"github.com/banzaicloud/pipeline/pkg/providers/azure"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
	"go.uber.org/cadence/client"
	corev1 "k8s.io/api/core/v1"
)

const pkeVersion = "0.4.6"
const MasterNodeTaint = pkgPKE.TaintKeyMaster + ":" + string(corev1.TaintEffectNoSchedule)

func MakeAzurePKEClusterCreator(logger logrus.FieldLogger, store pke.AzurePKEClusterStore, workflowClient client.Client, pipelineExternalURL string) AzurePKEClusterCreator {
	return AzurePKEClusterCreator{
		logger:              logger,
		store:               store,
		workflowClient:      workflowClient,
		pipelineExternalURL: pipelineExternalURL,
		paramsPreparer:      MakeAzurePKEClusterCreationParamsPreparer(logger),
	}
}

// AzurePKEClusterCreator creates new PKE-on-Azure clusters
type AzurePKEClusterCreator struct {
	logger              logrus.FieldLogger
	paramsPreparer      AzurePKEClusterCreationParamsPreparer
	store               pke.AzurePKEClusterStore
	workflowClient      client.Client
	pipelineExternalURL string
	secrets             interface {
		Get(organizationID uint, secretID string) (*secret.SecretItemResponse, error)
		Store(organizationID uint, request *secret.CreateSecretRequest) (string, error)
	}
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

func (np NodePool) hasRole(role pkgPKE.Role) bool {
	for _, r := range np.Roles {
		if r == string(role) {
			return true
		}
	}
	return false
}

type Subnet struct {
	Name string
	CIDR string
}

// AzurePKEClusterCreationParams defines parameters for PKE-on-Azure cluster creation
type AzurePKEClusterCreationParams struct {
	CreatedBy      uint
	Features       []intCluster.Feature
	Kubernetes     intPKE.Kubernetes
	Name           string
	Network        VirtualNetwork
	NodePools      []NodePool
	OrganizationID uint
	ResourceGroup  string
	ScaleOptions   pkgCluster.ScaleOptions
	SecretID       string
	SSHSecretID    string
}

// Create
func (cc AzurePKEClusterCreator) Create(ctx context.Context, params AzurePKEClusterCreationParams) (cl pke.PKEOnAzureCluster, err error) {
	if err = cc.paramsPreparer.Prepare(ctx, &params); err != nil {
		return
	}
	nodePools := make([]pke.NodePool, len(params.NodePools))
	for i, np := range params.NodePools {
		nodePools[i] = pke.NodePool{
			Autoscaling:  np.Autoscaling,
			CreatedBy:    np.CreatedBy,
			DesiredCount: uint(np.Count),
			InstanceType: np.InstanceType,
			Labels:       np.Labels,
			Max:          uint(np.Max),
			Min:          uint(np.Min),
			Name:         np.Name,
			Roles:        np.Roles,
			Subnet: pke.Subnetwork{
				Name: np.Subnet.Name,
			},
			Zones: np.Zones,
		}
	}
	createParams := pke.CreateParams{
		Name:               params.Name,
		OrganizationID:     params.OrganizationID,
		CreatedBy:          params.CreatedBy,
		Location:           params.Network.Location,
		SecretID:           params.SecretID,
		SSHSecretID:        params.SSHSecretID,
		RBAC:               params.Kubernetes.RBAC,
		ScaleOptions:       params.ScaleOptions,
		ResourceGroupName:  params.ResourceGroup,
		NodePools:          nodePools,
		VirtualNetworkName: params.Network.Name,
		KubernetesVersion:  params.Kubernetes.Version,
	}
	cl, err = cc.store.Create(createParams)
	if err != nil {
		return
	}

	sir, err := secret.Store.Get(params.OrganizationID, params.SecretID)
	if err != nil {
		cc.handleError(cl.ID, err)
		return
	}
	tenantID := sir.GetValue(pkgSecret.AzureTenantID)

	postHooks := make(pkgCluster.PostHooks, len(params.Features))
	for _, f := range params.Features {
		postHooks[f.Kind] = f.Params
	}
	{
		var commonCluster cluster.CommonCluster
		commonCluster, err = commoncluster.MakeCommonClusterGetter(secret.Store, cc.store).GetByID(cl.ID)
		if err != nil {
			cc.handleError(cl.ID, err)
			return
		}
		var labelsMap map[string]map[string]string
		labelsMap, err = cluster.GetDesiredLabelsForCluster(ctx, commonCluster, nil, false)
		if err != nil {
			cc.handleError(cl.ID, err)
			return
		}

		postHooks[pkgCluster.SetupNodePoolLabelsSet] = cluster.NodePoolLabelParam{
			Labels: labelsMap,
		}
	}

	sshKeyPair, err := GetOrCreateSSHKeyPair(cl, cc.secrets, cc.store)
	if err = emperror.Wrap(err, "failed to get or create SSH key pair"); err != nil {
		cc.handleError(cl.ID, err)
		return
	}

	tf := nodePoolTemplateFactory{
		ClusterID:           cl.ID,
		ClusterName:         cl.Name,
		KubernetesVersion:   cl.Kubernetes.Version,
		Location:            cl.Location,
		OrganizationID:      cl.OrganizationID,
		PipelineExternalURL: cc.pipelineExternalURL,
		ResourceGroupName:   cl.ResourceGroup.Name,
		SingleNodePool:      len(cl.NodePools) == 1,
		SSHPublicKey:        sshKeyPair.PublicKeyData,
		TenantID:            tenantID,
		VirtualNetworkName:  cl.VirtualNetwork.Name,
	}

	subnets := make(map[string]workflow.SubnetTemplate)
	vmssTemplates := make([]workflow.VirtualMachineScaleSetTemplate, len(params.NodePools))
	roleAssignmentTemplates := make([]workflow.RoleAssignmentTemplate, 0, len(params.NodePools))
	for i, np := range params.NodePools {
		vmsst, snt, rats := tf.getTemplates(np)
		vmssTemplates[i] = vmsst
		subnets[snt.Name] = snt
		roleAssignmentTemplates = append(roleAssignmentTemplates, rats...)
	}

	subnetTemplates := make([]workflow.SubnetTemplate, 0, len(subnets))
	for _, s := range subnets {
		subnetTemplates = append(subnetTemplates, s)
	}

	input := workflow.CreateClusterWorkflowInput{
		ClusterID:         cl.ID,
		ClusterName:       params.Name,
		OrganizationID:    params.OrganizationID,
		ResourceGroupName: params.ResourceGroup,
		SecretID:          params.SecretID,
		VirtualNetworkTemplate: workflow.VirtualNetworkTemplate{
			Name: params.Network.Name,
			CIDRs: []string{
				params.Network.CIDR,
			},
			Location: params.Network.Location,
			Subnets:  subnetTemplates,
		},
		LoadBalancerTemplate: workflow.LoadBalancerTemplate{
			Name:                           params.Name, // LB name must match the value passed to pke install master --kubernetes-cluster-name
			Location:                       params.Network.Location,
			SKU:                            "Standard",
			BackendAddressPoolName:         "backend-address-pool",
			OutboundBackendAddressPoolName: "outbound-backend-address-pool",
			InboundNATPoolName:             "ssh-inbound-nat-pool",
		},
		PublicIPAddress: workflow.PublicIPAddress{
			Location: params.Network.Location,
			Name:     params.Name + "-pip-in",
			SKU:      "Standard",
		},
		RoleAssignmentTemplates: roleAssignmentTemplates,
		RouteTable: workflow.RouteTable{
			Name:     params.Name + "-route-table",
			Location: params.Network.Location,
		},
		SecurityGroups: []workflow.SecurityGroup{
			{
				Name:     params.Name + "-master-nsg",
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
				Name:     params.Name + "-worker-nsg",
				Location: params.Network.Location,
				Rules:    []workflow.SecurityRule{},
			},
		},
		VirtualMachineScaleSetTemplates: vmssTemplates,
		PostHooks:                       postHooks,
	}
	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: 40 * time.Minute, // TODO: lower timeout
	}

	wfexec, err := cc.workflowClient.StartWorkflow(ctx, workflowOptions, workflow.CreateClusterWorkflowName, input)
	if err != nil {
		cc.handleError(cl.ID, err)
		return
	}

	if err = cc.store.SetActiveWorkflowID(cl.ID, wfexec.ID); err != nil {
		cc.logger.WithField("clusterID", cl.ID).WithField("workflowID", wfexec.ID).Error("failed to set active workflow ID", err)
		return
	}

	return
}

func (cc AzurePKEClusterCreator) handleError(clusterID uint, err error) error {
	return handleClusterError(cc.logger, cc.store, pkgCluster.Error, clusterID, err)
}

// AzurePKEClusterCreationParamsPreparer implements AzurePKEClusterCreationParams preparation
type AzurePKEClusterCreationParamsPreparer struct {
	k8sPreparer intPKE.KubernetesPreparer
	logger      logrus.FieldLogger
}

// MakeAzurePKEClusterCreationParamsPreparer returns an instance of AzurePKEClusterCreationParamsPreparer
func MakeAzurePKEClusterCreationParamsPreparer(logger logrus.FieldLogger) AzurePKEClusterCreationParamsPreparer {
	return AzurePKEClusterCreationParamsPreparer{
		k8sPreparer: intPKE.MakeKubernetesPreparer(logger, "Kubernetes"),
		logger:      logger,
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

	_, network, err := net.ParseCIDR(params.Network.CIDR)
	if err != nil {
		return emperror.Wrap(err, "failed to parse network CIDR")
	}
	if err := p.getNodePoolsPreparer(*network).Prepare(ctx, params.NodePools); err != nil {
		return emperror.Wrap(err, "failed to prepare node pools")
	}

	return nil
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

func (p AzurePKEClusterCreationParamsPreparer) getNodePoolsPreparer(network net.IPNet) NodePoolsPreparer {
	return NodePoolsPreparer{
		logger:    p.logger,
		namespace: "NodePools",
		dataProvider: clusterCreatorNodePoolPreparerDataProvider{
			network: network,
		},
	}
}

type clusterCreatorNodePoolPreparerDataProvider struct {
	network net.IPNet
}

func (p clusterCreatorNodePoolPreparerDataProvider) getExistingNodePoolByName(ctx context.Context, nodePoolName string) (pke.NodePool, error) {
	return pke.NodePool{}, notExistsYetError{}
}

func (p clusterCreatorNodePoolPreparerDataProvider) getSubnetCIDR(ctx context.Context, nodePool pke.NodePool) (string, error) {
	return "", notExistsYetError{}
}

func (p clusterCreatorNodePoolPreparerDataProvider) getVirtualNetworkAddressRange(ctx context.Context) (net.IPNet, error) {
	return p.network, nil
}

type notExistsYetError struct{}

func (notExistsYetError) Error() string {
	return "this resource does not exist yet"
}

func (notExistsYetError) NotFound() bool {
	return true
}

const masterUserDataScriptTemplate = `#!/bin/sh
export PRIVATE_IP=$(hostname -I | cut -d" " -f 1)
until curl -v https://banzaicloud.com/downloads/pke/pke-{{ .PKEVersion }} -o /usr/local/bin/pke; do sleep 10; done
chmod +x /usr/local/bin/pke
export PATH=$PATH:/usr/local/bin/

pke install master --pipeline-url="{{ .PipelineURL }}" \
--pipeline-token="{{ .PipelineToken }}" \
--pipeline-org-id={{ .OrgID }} \
--pipeline-cluster-id={{ .ClusterID}} \
--kubernetes-cluster-name={{ .ClusterName }} \
--pipeline-nodepool={{ .NodePoolName }} \
--taints={{ .Taints }} \
--kubernetes-cloud-provider=azure \
--azure-tenant-id={{ .TenantID }} \
--azure-subnet-name={{ .SubnetName }} \
--azure-security-group-name={{ .NSGName }} \
--azure-vnet-name={{ .VnetName }} \
--azure-vnet-resource-group={{ .VnetResourceGroupName }} \
--azure-vm-type=vmss \
--azure-loadbalancer-sku=standard \
--azure-route-table-name={{ .RouteTableName }} \
--azure-storage-kind managed \
--kubernetes-advertise-address=$PRIVATE_IP:6443 \
--kubernetes-api-server={{ .PublicAddress }}:6443 \
--kubernetes-infrastructure-cidr={{ .InfraCIDR }} \
--kubernetes-version={{ .KubernetesVersion }} \
--kubernetes-api-server-cert-sans={{ .PublicAddress }}`

const workerUserDataScriptTemplate = `#!/bin/sh
until curl -v https://banzaicloud.com/downloads/pke/pke-{{ .PKEVersion }} -o /usr/local/bin/pke; do sleep 10; done
chmod +x /usr/local/bin/pke
export PATH=$PATH:/usr/local/bin/

pke install worker --pipeline-url="{{ .PipelineURL }}" \
--pipeline-token="{{ .PipelineToken }}" \
--pipeline-org-id={{ .OrgID }} \
--pipeline-cluster-id={{ .ClusterID}} \
--pipeline-nodepool={{ .NodePoolName }} \
--taints={{ .Taints }} \
--kubernetes-cloud-provider=azure \
--azure-tenant-id={{ .TenantID }} \
--azure-subnet-name={{ .SubnetName }} \
--azure-security-group-name={{ .NSGName }} \
--azure-vnet-name={{ .VnetName }} \
--azure-vnet-resource-group={{ .VnetResourceGroupName }} \
--azure-vm-type=vmss \
--azure-loadbalancer-sku=standard \
--azure-route-table-name={{ .RouteTableName }} \
--kubernetes-api-server={{ .PublicAddress }}:6443 \
--kubernetes-infrastructure-cidr={{ .InfraCIDR }} \
--kubernetes-version={{ .KubernetesVersion }} \
--kubernetes-pod-network-cidr=""`
