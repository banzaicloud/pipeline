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
	"net/http"
	"strings"
	"time"

	"emperror.dev/errors"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"go.uber.org/cadence/client"
	corev1 "k8s.io/api/core/v1"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	intPKE "github.com/banzaicloud/pipeline/internal/pke"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke/driver/commoncluster"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke/workflow"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgPKE "github.com/banzaicloud/pipeline/pkg/cluster/pke"
	pkgAzure "github.com/banzaicloud/pipeline/pkg/providers/azure"
	"github.com/banzaicloud/pipeline/secret"
)

const pkeVersion = "0.4.14"
const MasterNodeTaint = pkgPKE.TaintKeyMaster + ":" + string(corev1.TaintEffectNoSchedule)

func MakeAzurePKEClusterCreator(
	config ClusterCreatorConfig,
	logger logrus.FieldLogger,
	organizations OrganizationStore,
	secrets ClusterCreatorSecretStore,
	store pke.AzurePKEClusterStore,
	workflowClient client.Client,
) AzurePKEClusterCreator {
	return AzurePKEClusterCreator{
		config:         config,
		logger:         logger,
		organizations:  organizations,
		secrets:        secrets,
		store:          store,
		workflowClient: workflowClient,
	}
}

// AzurePKEClusterCreator creates new PKE-on-Azure clusters
type AzurePKEClusterCreator struct {
	config         ClusterCreatorConfig
	logger         logrus.FieldLogger
	organizations  OrganizationStore
	secrets        ClusterCreatorSecretStore
	store          pke.AzurePKEClusterStore
	workflowClient client.Client
}

type OrganizationStore interface {
	Get(ctx context.Context, id uint) (auth.Organization, error)
}

type ClusterCreatorSecretStore interface {
	secretStore

	GetByName(organizationID uint, secretName string) (*secret.SecretItemResponse, error)
}

type ClusterCreatorConfig struct {
	OIDCIssuerURL               string
	PipelineExternalURL         string
	PipelineExternalURLInsecure bool
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

func (np NodePool) toPke() (pnp pke.NodePool) {
	pnp.Autoscaling = np.Autoscaling
	pnp.CreatedBy = np.CreatedBy
	pnp.DesiredCount = uint(np.Count)
	pnp.InstanceType = np.InstanceType
	pnp.Max = uint(np.Max)
	pnp.Min = uint(np.Min)
	pnp.Name = np.Name
	pnp.Roles = np.Roles
	pnp.Subnet = pke.Subnetwork{Name: np.Subnet.Name}
	pnp.Zones = np.Zones
	return
}

type Subnet struct {
	Name string
	CIDR string
}

// AzurePKEClusterCreationParams defines parameters for PKE-on-Azure cluster creation
type AzurePKEClusterCreationParams struct {
	CreatedBy             uint
	Features              []intCluster.Feature
	Kubernetes            intPKE.Kubernetes
	Name                  string
	Network               VirtualNetwork
	NodePools             []NodePool
	OrganizationID        uint
	ResourceGroup         string
	ScaleOptions          pkgCluster.ScaleOptions
	SecretID              string
	SSHSecretID           string
	HTTPProxy             intPKE.HTTPProxy
	AccessPoints          pke.AccessPoints
	APIServerAccessPoints pke.APIServerAccessPoints
}

// Create
func (cc AzurePKEClusterCreator) Create(ctx context.Context, params AzurePKEClusterCreationParams) (cl pke.PKEOnAzureCluster, err error) {
	sir, err := cc.secrets.Get(params.OrganizationID, params.SecretID)
	if err = errors.WrapIf(err, "failed to get secret"); err != nil {
		return
	}

	conn, err := pkgAzure.NewCloudConnection(&azure.PublicCloud, pkgAzure.NewCredentials(sir.Values))
	if err = errors.WrapIf(err, "failed to create new Azure cloud connection"); err != nil {
		return
	}

	if err = MakeAzurePKEClusterCreationParamsPreparer(conn, cc.logger).Prepare(ctx, &params); err != nil {
		return
	}

	routeTable := workflow.RouteTable{
		Name:     pke.GetRouteTableName(params.Name),
		Location: params.Network.Location,
	}

	sn, err := conn.GetSubnetsClient().Get(ctx, params.ResourceGroup, params.Network.Name, params.NodePools[0].Subnet.Name, "routeTable")
	if err = errors.WrapIf(err, "failed to get subnet"); err != nil && sn.StatusCode != http.StatusNotFound {
		_ = cc.handleError(cl.ID, err)
		return
	}

	if sn.StatusCode == http.StatusOK && sn.SubnetPropertiesFormat != nil && sn.RouteTable != nil && sn.RouteTable.ID != nil {
		routeTable = workflow.RouteTable{
			ID:       to.String(sn.RouteTable.ID),
			Name:     to.String(sn.RouteTable.Name),
			Location: to.String(sn.RouteTable.Location),
		}
	}

	nodePools := make([]pke.NodePool, len(params.NodePools))
	for i, np := range params.NodePools {
		nodePools[i] = pke.NodePool{
			Autoscaling:  np.Autoscaling,
			CreatedBy:    np.CreatedBy,
			DesiredCount: uint(np.Count),
			InstanceType: np.InstanceType,
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
		Name:                  params.Name,
		OrganizationID:        params.OrganizationID,
		CreatedBy:             params.CreatedBy,
		Location:              params.Network.Location,
		SecretID:              params.SecretID,
		SSHSecretID:           params.SSHSecretID,
		RBAC:                  params.Kubernetes.RBAC,
		OIDC:                  params.Kubernetes.OIDC.Enabled,
		ScaleOptions:          params.ScaleOptions,
		ResourceGroupName:     params.ResourceGroup,
		NodePools:             nodePools,
		VirtualNetworkName:    params.Network.Name,
		KubernetesVersion:     params.Kubernetes.Version,
		HTTPProxy:             params.HTTPProxy,
		AccessPoints:          params.AccessPoints,
		APIServerAccessPoints: params.APIServerAccessPoints,
	}
	cl, err = cc.store.Create(createParams)
	if err != nil {
		return
	}

	tenantID := sir.GetValue(secrettype.AzureTenantID)

	postHooks := make(pkgCluster.PostHooks, len(params.Features))
	for _, f := range params.Features {
		postHooks[f.Kind] = f.Params
	}
	{
		var commonCluster cluster.CommonCluster
		commonCluster, err = commoncluster.MakeCommonClusterGetter(cc.secrets, cc.store).GetByID(cl.ID)
		if err != nil {
			_ = cc.handleError(cl.ID, err)
			return
		}
		nodePoolStatuses := make(map[string]*pkgCluster.NodePoolStatus, len(params.NodePools))
		for _, np := range params.NodePools {
			nodePoolStatuses[np.Name] = &pkgCluster.NodePoolStatus{
				Autoscaling:  np.Autoscaling,
				Count:        np.Count,
				InstanceType: np.InstanceType,
				MinCount:     np.Min,
				MaxCount:     np.Max,
				Labels:       np.Labels,
			}
		}
		var labelsMap map[string]map[string]string
		labelsMap, err = cluster.GetDesiredLabelsForCluster(ctx, commonCluster, nodePoolStatuses, false)
		if err != nil {
			_ = cc.handleError(cl.ID, err)
			return
		}

		postHooks[pkgCluster.SetupNodePoolLabelsSet] = cluster.NodePoolLabelParam{
			Labels: labelsMap,
		}
	}

	sshKeyPair, err := GetOrCreateSSHKeyPair(cl, cc.secrets, cc.store)
	if err = errors.WrapIf(err, "failed to get or create SSH key pair"); err != nil {
		_ = cc.handleError(cl.ID, err)
		return
	}

	tf := nodePoolTemplateFactory{
		ClusterID:                   cl.ID,
		ClusterName:                 cl.Name,
		KubernetesVersion:           cl.Kubernetes.Version,
		Location:                    cl.Location,
		NoProxy:                     strings.Join(cl.HTTPProxy.Exceptions, ","),
		OrganizationID:              cl.OrganizationID,
		PipelineExternalURL:         cc.config.PipelineExternalURL,
		PipelineExternalURLInsecure: cc.config.PipelineExternalURLInsecure,
		ResourceGroupName:           cl.ResourceGroup.Name,
		RouteTableName:              routeTable.Name,
		SingleNodePool:              len(cl.NodePools) == 1,
		SSHPublicKey:                sshKeyPair.PublicKeyData,
		TenantID:                    tenantID,
		VirtualNetworkName:          cl.VirtualNetwork.Name,
	}

	if cl.Kubernetes.OIDC.Enabled {
		tf.OIDCIssuerURL = cc.config.OIDCIssuerURL
		tf.OIDCClientID = cl.UID
	}

	subnets := make(map[string]workflow.SubnetTemplate)
	vmssTemplates := make([]workflow.VirtualMachineScaleSetTemplate, len(params.NodePools))
	roleAssignmentTemplates := make([]workflow.RoleAssignmentTemplate, 0, len(params.NodePools))
	var masterNodesSubnetName string
	for i, np := range params.NodePools {
		vmsst, snt, rats := tf.getTemplates(np)
		vmssTemplates[i] = vmsst
		subnets[snt.Name] = snt
		roleAssignmentTemplates = append(roleAssignmentTemplates, rats...)

		if np.hasRole(pkgPKE.RoleMaster) {
			masterNodesSubnetName = snt.Name
		}
	}

	subnetTemplates := make([]workflow.SubnetTemplate, 0, len(subnets))
	for _, s := range subnets {
		subnetTemplates = append(subnetTemplates, s)
	}

	var pip workflow.PublicIPAddress
	loadBalancerTemplates := make([]workflow.LoadBalancerTemplate, len(params.AccessPoints))
	for i, accessPoint := range params.AccessPoints {
		var subnetName, publicIPAddressName, outboundBackendAddressPoolName, backendAddressPoolName, inboundNATPoolName, lbName string

		if accessPoint.Name == "private" {
			lbName = pke.GetLoadBalancerName(params.Name) + "-internal"

			// private access point implemented through internal LB which requires a subnet
			// use master node's subnet for the internal LB
			subnetName = masterNodesSubnetName

			if params.APIServerAccessPoints.Exists("private") {
				// add master nodes LB backend address pool
				backendAddressPoolName = pke.GetBackendAddressPoolName()
			}
		} else {
			lbName = pke.GetLoadBalancerName(params.Name)
			// backend pool for ensuring outbound connectivity to the Internet through public Standard LB
			outboundBackendAddressPoolName = pke.GetOutboundBackendAddressPoolName()

			if params.APIServerAccessPoints.Exists("public") {
				// if API server is exposed through public end point, set up INAT
				// through public LB to be able to ssh to master nodes
				inboundNATPoolName = pke.GetInboundNATPoolName()

				// add master nodes LB backend address pool
				backendAddressPoolName = pke.GetBackendAddressPoolName()
			}

			// Public IP address for public LB
			publicIPAddressName = pke.GetPublicIPAddressName(params.Name)
			pip = workflow.PublicIPAddress{
				Location: params.Network.Location,
				Name:     publicIPAddressName,
				SKU:      "Standard",
			}
		}

		loadBalancerTemplates[i] = workflow.LoadBalancerTemplate{
			Name:                           lbName,
			Location:                       params.Network.Location,
			SKU:                            "Standard",
			BackendAddressPoolName:         backendAddressPoolName,
			OutboundBackendAddressPoolName: outboundBackendAddressPoolName,
			InboundNATPoolName:             inboundNATPoolName,
			SubnetName:                     subnetName,
			PublicIPAddressName:            publicIPAddressName,
		}
	}

	org, err := cc.organizations.Get(ctx, params.OrganizationID)
	if err != nil {
		return cl, errors.WrapIf(err, "failed to get organization")
	}

	input := workflow.CreateClusterWorkflowInput{
		ClusterID:         cl.ID,
		ClusterName:       cl.Name,
		ClusterUID:        cl.UID,
		OrganizationID:    org.ID,
		OrganizationName:  org.Name,
		ResourceGroupName: params.ResourceGroup,
		SecretID:          params.SecretID,
		OIDCEnabled:       cl.Kubernetes.OIDC.Enabled,
		VirtualNetworkTemplate: workflow.VirtualNetworkTemplate{
			Name: params.Network.Name,
			CIDRs: []string{
				params.Network.CIDR,
			},
			Location: params.Network.Location,
			Subnets:  subnetTemplates,
		},
		LoadBalancerTemplates:   loadBalancerTemplates,
		PublicIPAddress:         pip,
		RoleAssignmentTemplates: roleAssignmentTemplates,
		RouteTable:              routeTable,
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
		HTTPProxy:                       cl.HTTPProxy,
		AccessPoints:                    params.AccessPoints,
		APIServerAccessPoints:           params.APIServerAccessPoints,
	}
	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: 40 * time.Minute, // TODO: lower timeout
	}

	wfexec, err := cc.workflowClient.StartWorkflow(ctx, workflowOptions, workflow.CreateClusterWorkflowName, input)
	if err != nil {
		_ = cc.handleError(cl.ID, err)
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
	connection  *pkgAzure.CloudConnection
	k8sPreparer intPKE.KubernetesPreparer
	logger      logrus.FieldLogger
}

// MakeAzurePKEClusterCreationParamsPreparer returns an instance of AzurePKEClusterCreationParamsPreparer
func MakeAzurePKEClusterCreationParamsPreparer(connection *pkgAzure.CloudConnection, logger logrus.FieldLogger) AzurePKEClusterCreationParamsPreparer {
	return AzurePKEClusterCreationParamsPreparer{
		connection:  connection,
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

	if len(params.AccessPoints) == 0 {
		params.AccessPoints = append(params.AccessPoints, pke.AccessPoint{Name: "public"})
		p.logger.Debug("access points not specified, defaulting to public")
	}
	if len(params.AccessPoints) > 2 {
		return validationError{"only private, public or both access points are allowed"}
	}

	for _, ap := range params.AccessPoints {
		if ap.Name != "private" && ap.Name != "public" {
			return validationError{"only private or public access points are allowed"}
		}
	}

	if params.APIServerAccessPoints == nil {
		params.APIServerAccessPoints = append(params.APIServerAccessPoints, "public")
		p.logger.Debug("API server access points not specified, defaulting to public")
	}

	if len(params.APIServerAccessPoints) > 2 {
		return validationErrorf("only private, public or both are allowed as API server access points")
	}
	for _, apiServerAp := range params.APIServerAccessPoints {
		if !params.AccessPoints.Exists(apiServerAp.GetName()) {
			return validationError{fmt.Sprintf("no access point is defined for API server access point %s", apiServerAp)}
		}
	}

	if err := p.k8sPreparer.Prepare(&params.Kubernetes); err != nil {
		return errors.WrapIf(err, "failed to prepare k8s network")
	}

	if err := p.getVNetPreparer(p.connection, params.Name, params.ResourceGroup).Prepare(ctx, &params.Network); err != nil {
		return errors.WrapIf(err, "failed to prepare cluster network")
	}

	_, network, err := net.ParseCIDR(params.Network.CIDR)
	if err != nil {
		return errors.WrapIf(err, "failed to parse network CIDR")
	}
	if err := p.getNodePoolsPreparer(clusterCreatorNodePoolPreparerDataProvider{
		resourceGroupName:  params.ResourceGroup,
		subnetsClient:      *p.connection.GetSubnetsClient(),
		virtualNetworkCIDR: *network,
		virtualNetworkName: params.Network.Name,
	}).Prepare(ctx, params.NodePools); err != nil {
		return errors.WrapIf(err, "failed to prepare node pools")
	}

	return nil
}

func (p AzurePKEClusterCreationParamsPreparer) getVNetPreparer(cloudConnection *pkgAzure.CloudConnection, clusterName, resourceGroupName string) VirtualNetworkPreparer {
	return VirtualNetworkPreparer{
		clusterName:       clusterName,
		connection:        cloudConnection,
		logger:            p.logger,
		namespace:         "Network",
		resourceGroupName: resourceGroupName,
	}
}

func (p AzurePKEClusterCreationParamsPreparer) getNodePoolsPreparer(dataProvider nodePoolsDataProvider) NodePoolsPreparer {
	return NodePoolsPreparer{
		logger:       p.logger,
		namespace:    "NodePools",
		dataProvider: dataProvider,
	}
}

type clusterCreatorNodePoolPreparerDataProvider struct {
	resourceGroupName  string
	subnetsClient      pkgAzure.SubnetsClient
	virtualNetworkCIDR net.IPNet
	virtualNetworkName string
}

func (p clusterCreatorNodePoolPreparerDataProvider) getExistingNodePools(ctx context.Context) ([]pke.NodePool, error) {
	return nil, nil
}

func (p clusterCreatorNodePoolPreparerDataProvider) getExistingNodePoolByName(ctx context.Context, nodePoolName string) (pke.NodePool, error) {
	return pke.NodePool{}, notExistsYetError{}
}

func (p clusterCreatorNodePoolPreparerDataProvider) getSubnetCIDR(ctx context.Context, subnetName string) (string, error) {
	return getSubnetCIDR(ctx, p.subnetsClient, p.resourceGroupName, p.virtualNetworkName, subnetName)
}

func (p clusterCreatorNodePoolPreparerDataProvider) getVirtualNetworkAddressRange(ctx context.Context) (net.IPNet, error) {
	return p.virtualNetworkCIDR, nil
}

const masterUserDataScriptTemplate = `#!/bin/sh
export PRIVATE_IP=$(hostname -I | cut -d" " -f 1)
until curl -v https://banzaicloud.com/downloads/pke/pke-{{ .PKEVersion }} -o /usr/local/bin/pke; do sleep 10; done
chmod +x /usr/local/bin/pke
export PATH=$PATH:/usr/local/bin/
export HTTP_PROXY="{{ .HttpProxy }}"
export HTTPS_PROXY="{{ .HttpsProxy }}"
export NO_PROXY="{{ .NoProxy }}"

pke install master --pipeline-url="{{ .PipelineURL }}" \
--pipeline-insecure="{{ .PipelineURLInsecure }}" \
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
--kubernetes-api-server={{ .ApiServerAddress }}:6443 \
--kubernetes-infrastructure-cidr={{ .InfraCIDR }} \
--kubernetes-version={{ .KubernetesVersion }} \
--kubernetes-master-mode={{ .KubernetesMasterMode }} \
--kubernetes-api-server-cert-sans={{ .ApiServerCertSans }}`

const workerUserDataScriptTemplate = `#!/bin/sh
until curl -v https://banzaicloud.com/downloads/pke/pke-{{ .PKEVersion }} -o /usr/local/bin/pke; do sleep 10; done
chmod +x /usr/local/bin/pke
export PATH=$PATH:/usr/local/bin/
export HTTP_PROXY="{{ .HttpProxy }}"
export HTTPS_PROXY="{{ .HttpsProxy }}"
export NO_PROXY="{{ .NoProxy }}"

pke install worker --pipeline-url="{{ .PipelineURL }}" \
--pipeline-insecure="{{ .PipelineURLInsecure }}" \
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
--kubernetes-api-server={{ .ApiServerAddress }}:6443 \
--kubernetes-infrastructure-cidr={{ .InfraCIDR }} \
--kubernetes-version={{ .KubernetesVersion }} \
--kubernetes-pod-network-cidr=""`
