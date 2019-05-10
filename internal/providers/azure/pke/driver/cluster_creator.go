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
	"strconv"
	"strings"
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
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/providers/azure"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/gofrs/uuid"
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

	var sshKeyPair *secret.SSHKeyPair
	if params.SSHSecretID == "" {
		var sshSecretID string
		sshKeyPair, sshSecretID, err = newSSHKeyPair(cl.OrganizationID, cl.ID, cl.Name, cl.UID)
		if err != nil {
			cc.handleError(cl.ID, err)
			return
		}
		if err = cc.store.SetSSHSecretID(cl.ID, sshSecretID); err != nil {
			cc.handleError(cl.ID, err)
			return
		}
	} else {
		sshKeyPair, err = getSSHKeyPair(cl.OrganizationID, params.SSHSecretID)
		if err != nil {
			cc.handleError(cl.ID, err)
			return
		}
	}
	sshPublicKey := sshKeyPair.PublicKeyData

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

	subnets := make(map[string]workflow.SubnetTemplate)
	vmssTemplates := make([]workflow.VirtualMachineScaleSetTemplate, len(params.NodePools))
	roleAssignmentTemplates := make([]workflow.RoleAssignmentTemplate, len(params.NodePools))
	npLen := len(params.NodePools)
	for i, np := range params.NodePools {
		var bapn string
		var inpn string
		var nsgn string
		var cnsgn string
		var azureRole string
		var taints string
		var userDataScriptTemplate string

		switch {
		case np.hasRole(pkgPKE.RoleMaster):
			bapn = "backend-address-pool"
			inpn = "ssh-inbound-nat-pool"
			nsgn = params.Name + "-master-nsg"
			azureRole = "Owner"

			if npLen > 1 {
				taints = MasterNodeTaint
			} else {
				taints = "," // do not taint single-nodepool cluster's master node
			}

			userDataScriptTemplate = masterUserDataScriptTemplate
		default:
			nsgn = params.Name + "-worker-nsg"
			azureRole = "Contributor"
			if npLen > 1 && np.hasRole(pkgPKE.RolePipelineSystem) {
				taints = fmt.Sprintf("%s=%s:%s", pkgCommon.NodePoolNameTaintKey, np.Name, corev1.TaintEffectPreferNoSchedule)
			}
			userDataScriptTemplate = workerUserDataScriptTemplate

		}
		cnsgn = nsgn
		if npLen > 1 {
			// Ingress traffic flow target. In case of multiple NSGs workers can only receive traffic.
			cnsgn = params.Name + "-worker-nsg"
		}

		subnets[np.Subnet.Name] = workflow.SubnetTemplate{
			Name:                     np.Subnet.Name,
			CIDR:                     np.Subnet.CIDR,
			NetworkSecurityGroupName: nsgn,
		}

		vmssName := pke.GetVMSSName(params.Name, np.Name)
		vmssTemplates[i] = workflow.VirtualMachineScaleSetTemplate{
			AdminUsername: "azureuser",
			Image: workflow.Image{
				Offer:     "CentOS-CI",
				Publisher: "OpenLogic",
				SKU:       "7-CI",
				Version:   "7.6.20190306",
			},
			InstanceCount:          uint(np.Count),
			InstanceType:           np.InstanceType,
			BackendAddressPoolName: bapn,
			InboundNATPoolName:     inpn,
			Location:               params.Network.Location,
			Name:                   vmssName,
			SSHPublicKey:           sshPublicKey,
			SubnetName:             np.Subnet.Name,
			UserDataScriptParams: map[string]string{
				"ClusterID":             strconv.FormatUint(uint64(cl.ID), 10),
				"ClusterName":           params.Name,
				"InfraCIDR":             np.Subnet.CIDR,
				"LoadBalancerSKU":       "standard",
				"NodePoolName":          np.Name,
				"Taints":                taints,
				"NSGName":               cnsgn,
				"OrgID":                 strconv.FormatUint(uint64(params.OrganizationID), 10),
				"PipelineURL":           cc.pipelineExternalURL,
				"PipelineToken":         "<not yet set>",
				"PKEVersion":            pkeVersion,
				"KubernetesVersion":     params.Kubernetes.Version,
				"PublicAddress":         "<not yet set>",
				"RouteTableName":        params.Name + "-route-table",
				"SubnetName":            np.Subnet.Name,
				"TenantID":              tenantID,
				"VnetName":              params.Network.Name,
				"VnetResourceGroupName": params.ResourceGroup,
			},
			UserDataScriptTemplate: userDataScriptTemplate,
			Zones:                  np.Zones,
		}

		roleAssignmentTemplates[i] = workflow.RoleAssignmentTemplate{
			Name:     uuid.Must(uuid.NewV1()).String(),
			VMSSName: vmssName,
			RoleName: azureRole,
		}
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
			Name:                   params.Name, // LB name must match the value passed to pke install master --kubernetes-cluster-name
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
	if clusterID != 0 && err != nil {
		if err := cc.store.SetStatus(clusterID, pkgCluster.Error, err.Error()); err != nil {
			cc.logger.Errorf("failed to set cluster error status: %s", err.Error())
		}
	}
	return err
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
		if nodePools[i].Subnet.Name == "" {
			if nodePools[i].Subnet.CIDR == "" {
				nodePools[i].Subnet.CIDR = fmt.Sprintf("10.0.%d.0/24", i)
			}
			nodePools[i].Subnet.Name = fmt.Sprintf("subnet-%d", i)
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
