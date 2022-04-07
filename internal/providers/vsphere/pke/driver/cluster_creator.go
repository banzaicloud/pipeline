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
	"strings"
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence/client"
	corev1 "k8s.io/api/core/v1"

	intPKE "github.com/banzaicloud/pipeline/internal/pke"
	"github.com/banzaicloud/pipeline/internal/providers/vsphere/pke"
	"github.com/banzaicloud/pipeline/internal/providers/vsphere/pke/driver/commoncluster"
	"github.com/banzaicloud/pipeline/internal/providers/vsphere/pke/workflow"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgPKE "github.com/banzaicloud/pipeline/pkg/cluster/pke"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/cluster"
	"github.com/banzaicloud/pipeline/src/secret"
)

const (
	pkeVersion      = "0.8.1"
	MasterNodeTaint = pkgPKE.TaintKeyMaster + ":" + string(corev1.TaintEffectNoSchedule)
)

func MakeVspherePKEClusterCreator(
	logger Logger,
	config ClusterConfig,
	k8sPreparer intPKE.KubernetesPreparer,
	organizations OrganizationStore,
	secrets ClusterCreatorSecretStore,
	store pke.ClusterStore,
	workflowClient client.Client,
) VspherePKEClusterCreator {
	return VspherePKEClusterCreator{
		logger:           logger,
		config:           config,
		creationPreparer: MakeVspherePKEClusterCreationParamsPreparer(logger, k8sPreparer, secrets),
		organizations:    organizations,
		secrets:          secrets,
		store:            store,
		workflowClient:   workflowClient,
	}
}

// VspherePKEClusterCreator creates new PKE-on-Vsphere clusters
type VspherePKEClusterCreator struct {
	logger           Logger
	config           ClusterConfig
	creationPreparer VspherePKEClusterCreationParamsPreparer
	organizations    OrganizationStore
	secrets          ClusterCreatorSecretStore
	store            pke.ClusterStore
	workflowClient   client.Client
}

type OrganizationStore interface {
	Get(ctx context.Context, id uint) (auth.Organization, error)
}

type ClusterCreatorSecretStore interface {
	secretStore

	GetByName(organizationID uint, secretName string) (*secret.SecretItemResponse, error)
}

type ClusterConfig struct {
	OIDCIssuerURL               string
	PipelineExternalURL         string
	PipelineExternalURLInsecure bool
}

type NodePool struct {
	CreatedBy     uint
	Name          string
	Roles         []string
	Labels        map[string]string
	Size          int
	AdminUsername string
	VCPU          int
	RAM           int // MiB
	TemplateName  string
}

func (np NodePool) InstanceType() string {
	return fmt.Sprintf("%dvcpu-%dmb", np.VCPU, np.RAM)
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
	pnp.Size = np.Size
	pnp.VCPU = np.VCPU
	pnp.RAM = np.RAM
	pnp.Name = np.Name
	pnp.Roles = np.Roles
	pnp.AdminUsername = np.AdminUsername
	pnp.TemplateName = np.TemplateName
	return
}

type Subnet struct {
	Name string
	CIDR string
}

// VspherePKEClusterCreationParams defines parameters for PKE-on-Vsphere cluster creation
type VspherePKEClusterCreationParams struct {
	CreatedBy           uint
	Name                string
	NodePools           []NodePool
	OrganizationID      uint
	SecretID            string
	StorageSecretID     string
	SSHSecretID         string
	HTTPProxy           intPKE.HTTPProxy
	ResourcePoolName    string
	FolderName          string
	DatastoreName       string
	Kubernetes          intPKE.Kubernetes
	ActiveWorkflowID    string
	LoadBalancerIPRange string
}

// Create
func (cc VspherePKEClusterCreator) Create(ctx context.Context, params VspherePKEClusterCreationParams) (cl pke.PKEOnVsphereCluster, err error) {
	var vsphereSecret *secret.SecretItemResponse
	vsphereSecret, err = cc.secrets.Get(params.OrganizationID, params.SecretID)
	if err = errors.WrapIf(err, "failed to get secret"); err != nil {
		return
	}

	defaultNodeTemplate := vsphereSecret.Values[secrettype.VsphereDefaultNodeTemplate]

	// TODO maybe check the connection here, OR don't fetch the secret at all

	if err = cc.creationPreparer.Prepare(ctx, &params); err != nil {
		return
	}

	nodePools := make([]pke.NodePool, len(params.NodePools))
	for i, np := range params.NodePools {
		nodePools[i] = pke.NodePool{
			CreatedBy:    np.CreatedBy,
			Name:         np.Name,
			Roles:        np.Roles,
			Size:         np.Size,
			VCPU:         np.VCPU,
			RAM:          np.RAM,
			TemplateName: np.TemplateName,
		}
		if nodePools[i].TemplateName == "" {
			nodePools[i].TemplateName = defaultNodeTemplate
		}
	}
	createParams := pke.CreateParams{
		Name:                params.Name,
		OrganizationID:      params.OrganizationID,
		CreatedBy:           params.CreatedBy,
		SecretID:            params.SecretID,
		StorageSecretID:     params.StorageSecretID,
		SSHSecretID:         params.SSHSecretID,
		RBAC:                params.Kubernetes.RBAC,
		OIDC:                params.Kubernetes.OIDC.Enabled,
		NodePools:           nodePools,
		HTTPProxy:           params.HTTPProxy,
		ResourcePoolName:    params.ResourcePoolName,
		FolderName:          params.FolderName,
		DatastoreName:       params.DatastoreName,
		Kubernetes:          params.Kubernetes,
		LoadBalancerIPRange: params.LoadBalancerIPRange,
	}
	cl, err = cc.store.Create(createParams)
	if err != nil {
		return
	}

	sshKeyPair, err := GetOrCreateSSHKeyPair(cl, cc.secrets, cc.store)
	if err = errors.WrapIf(err, "failed to get or create SSH key pair"); err != nil {
		_ = cc.handleError(cl.ID, err)
		return
	}

	tf := nodeTemplateFactory{
		ClusterID:                   cl.ID,
		ClusterName:                 cl.Name,
		KubernetesVersion:           cl.Kubernetes.Version,
		NoProxy:                     strings.Join(cl.HTTPProxy.Exceptions, ","),
		OrganizationID:              cl.OrganizationID,
		PipelineExternalURL:         cc.config.PipelineExternalURL,
		PipelineExternalURLInsecure: cc.config.PipelineExternalURLInsecure,
		SingleNodePool:              len(cl.NodePools) == 1,
		SSHPublicKey:                sshKeyPair.PublicKeyData,
		LoadBalancerIPRange:         cl.LoadBalancerIPRange,
	}

	if cl.Kubernetes.OIDC.Enabled {
		tf.OIDCIssuerURL = cc.config.OIDCIssuerURL
		tf.OIDCClientID = cl.UID
	}

	var nodes []workflow.Node
	for _, np := range cl.NodePools {
		for i := 1; i <= np.Size; i++ {
			nodes = append(nodes, tf.getNode(np, i))
		}
	}

	org, err := cc.organizations.Get(ctx, cl.OrganizationID)
	if err != nil {
		return cl, errors.WrapIf(err, "failed to get organization")
	}

	var labelsMap map[string]map[string]string
	{
		var commonCluster cluster.CommonCluster
		commonCluster, err = commoncluster.MakeCommonClusterGetter(cc.secrets, cc.store).GetByID(cl.ID)
		if err != nil {
			_ = cc.handleError(cl.ID, err)
			return
		}

		nodePoolLabels := make([]cluster.NodePoolLabels, 0)
		for _, np := range params.NodePools {
			nodePoolLabels = append(nodePoolLabels, cluster.NodePoolLabels{
				NodePoolName: np.Name,
				Existing:     false,
				// TODO setup instance name, memory, vcpu
				InstanceType: np.InstanceType(),
				CustomLabels: np.Labels,
			})
		}

		labelsMap, err = cluster.GetDesiredLabelsForCluster(ctx, commonCluster, nodePoolLabels)
		if err != nil {
			_ = cc.handleError(cl.ID, err)
			return
		}
	}

	input := workflow.CreateClusterWorkflowInput{
		ClusterID:        cl.ID,
		ClusterName:      cl.Name,
		ClusterUID:       cl.UID,
		OrganizationID:   cl.OrganizationID,
		OrganizationName: org.Name,
		SecretID:         cl.SecretID,
		StorageSecretID:  cl.StorageSecretID,
		OIDCEnabled:      cl.Kubernetes.OIDC.Enabled,
		Nodes:            nodes,
		HTTPProxy:        cl.HTTPProxy,
		NodePoolLabels:   labelsMap,
		ResourcePoolName: cl.ResourcePool,
		DatastoreName:    cl.Datastore,
		FolderName:       cl.Folder,
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
		_ = cc.handleError(cl.ID, err)
		return
	}

	return
}

func (cc VspherePKEClusterCreator) handleError(clusterID uint, err error) error {
	return handleClusterError(cc.logger, cc.store, pkgCluster.Error, clusterID, err)
}

// VspherePKEClusterCreationParamsPreparer implements VspherePKEClusterCreationParams preparation
type VspherePKEClusterCreationParamsPreparer struct {
	k8sPreparer intPKE.KubernetesPreparer
	logger      Logger
	secrets     ClusterCreatorSecretStore
}

// MakeVspherePKEClusterCreationParamsPreparer returns an instance of VspherePKEClusterCreationParamsPreparer
func MakeVspherePKEClusterCreationParamsPreparer(logger Logger, k8sPreparer intPKE.KubernetesPreparer, secrets ClusterCreatorSecretStore) VspherePKEClusterCreationParamsPreparer {
	return VspherePKEClusterCreationParamsPreparer{
		k8sPreparer: k8sPreparer,
		logger:      logger,
		secrets:     secrets,
	}
}

// Prepare validates and provides defaults for VspherePKEClusterCreationParams fields
func (p VspherePKEClusterCreationParamsPreparer) Prepare(ctx context.Context, params *VspherePKEClusterCreationParams) error {
	if params.Name == "" {
		return validationErrorf("Name cannot be empty")
	}
	if params.OrganizationID == 0 {
		return validationErrorf("OrganizationID cannot be 0")
	}

	_, err := auth.GetOrganizationById(params.OrganizationID)
	if err != nil {
		return validationErrorf("OrganizationID cannot be found %s", err.Error())
	}

	// validate secretID
	if params.SecretID == "" {
		return validationErrorf("SecretID cannot be empty")
	}
	if err := p.verifySecretIsOfType(params.OrganizationID, params.SecretID, secrettype.Vsphere); err != nil {
		return err
	}

	// validate storageSecretID if present
	if err := p.verifySecretIsOfType(params.OrganizationID, params.StorageSecretID, secrettype.Vsphere); err != nil {
		return err
	}

	// validate SSH secret ID if present
	if err := p.verifySecretIsOfType(params.OrganizationID, params.SSHSecretID, secrettype.SSHSecretType); err != nil {
		return err
	}

	if err := p.k8sPreparer.Prepare(&params.Kubernetes); err != nil {
		return errors.WrapIf(err, "failed to prepare k8s network")
	}

	if err := p.getNodePoolsPreparer(clusterCreatorNodePoolPreparerDataProvider{}).Prepare(ctx, params.NodePools); err != nil {
		return errors.WrapIf(err, "failed to prepare node pools")
	}

	return nil
}

func (p VspherePKEClusterCreationParamsPreparer) verifySecretIsOfType(orgID uint, secretID string, secretType string) error {
	if secretID == "" {
		return nil
	}
	secret, err := p.secrets.Get(orgID, secretID)
	if err != nil {
		return validationErrorf("failed to get secret %s", secretID)
	}
	if secret.Type != secretType {
		return validationErrorf("%s should be of type VSphere", secretID)
	}
	return nil
}

func (p VspherePKEClusterCreationParamsPreparer) getNodePoolsPreparer(dataProvider nodePoolsDataProvider) NodePoolsPreparer {
	return NodePoolsPreparer{
		logger:       p.logger,
		dataProvider: dataProvider,
	}
}

type clusterCreatorNodePoolPreparerDataProvider struct {
}

func (p clusterCreatorNodePoolPreparerDataProvider) getExistingNodePools(ctx context.Context) ([]pke.NodePool, error) {
	return nil, nil
}

func (p clusterCreatorNodePoolPreparerDataProvider) getExistingNodePoolByName(ctx context.Context, nodePoolName string) (pke.NodePool, error) {
	return pke.NodePool{}, notExistsYetError{}
}

const masterUserDataScriptTemplate = `
set -x
{{ if .HttpProxy }}export HTTP_PROXY="{{ .HttpProxy }}"{{ end }}
{{ if .HttpsProxy }}export HTTPS_PROXY="{{ .HttpsProxy }}"{{ end }}
{{ if .NoProxy }}export NO_PROXY="{{ .NoProxy }}"{{ end }}

PRIVATE_IP=$(hostname -I | cut -d" " -f 1)
PUBLIC_ADDRESS="{{ if .PublicAddress }}{{ .PublicAddress }}{{ else }}$PRIVATE_IP{{ end }}"

export PATH=$PATH:/usr/local/bin/
if ! command -v pke > /dev/null 2>&1; then
	until curl -v https://banzaicloud.com/downloads/pke/pke-{{ .PKEVersion }} -o /usr/local/bin/pke; do sleep 10; done
	chmod +x /usr/local/bin/pke
fi

if [ -r /etc/banzaicloud/pke.rc ]; then . /etc/banzaicloud/pke.rc; fi

pke install master --pipeline-url="{{ .PipelineURL }}" \
--pipeline-insecure="{{ .PipelineURLInsecure }}" \
--pipeline-token="{{ .PipelineToken }}" \
--pipeline-org-id={{ .OrgID }} \
--pipeline-cluster-id={{ .ClusterID}} \
--kubernetes-cluster-name={{ .ClusterName }} \
--pipeline-nodepool={{ .NodePoolName }} \
--taints={{ .Taints }} \
--kubernetes-advertise-address=$PRIVATE_IP:6443 \
--kubernetes-api-server=$PUBLIC_ADDRESS:6443 \
--kubernetes-infrastructure-cidr=$PRIVATE_IP/32 \
--kubernetes-version={{ .KubernetesVersion }} \
--kubernetes-master-mode={{ .KubernetesMasterMode }} \
--kubernetes-api-server-cert-sans="${PUBLIC_ADDRESS}" \
--kubernetes-cloud-provider=vsphere \
--vsphere-server="{{ .VCenterServer }}" \
--vsphere-port={{ .VCenterPort }} \
--vsphere-fingerprint="{{ .VCenterFingerprint }}" \
--vsphere-datacenter="{{ .Datacenter }}" \
--vsphere-datastore="{{ .Datastore }}" \
--vsphere-resourcepool="{{ .ResourcePool }}" \
--vsphere-folder="{{ .Folder }}" \
--vsphere-username="{{ .Username }}" \
--vsphere-password="{{ .Password }}" \
--lb-range="{{ .LoadBalancerIPRange }}" \
${PKE_EXTRA_ARGS:-}`

const workerUserDataScriptTemplate = `
set -x
{{ if .HttpProxy }}export HTTP_PROXY="{{ .HttpProxy }}"{{ end }}
{{ if .HttpsProxy }}export HTTPS_PROXY="{{ .HttpsProxy }}"{{ end }}
{{ if .NoProxy }}export NO_PROXY="{{ .NoProxy }}"{{ end }}

export PATH=$PATH:/usr/local/bin/
if ! command -v pke > /dev/null 2>&1; then
until curl -v https://banzaicloud.com/downloads/pke/pke-{{ .PKEVersion }} -o /usr/local/bin/pke; do sleep 10; done
chmod +x /usr/local/bin/pke
fi

PRIVATE_IP=$(hostname -I | cut -d" " -f 1)

if [ -r /etc/banzaicloud/pke.rc ]; then . /etc/banzaicloud/pke.rc; fi

pke install worker --pipeline-url="{{ .PipelineURL }}" \
--pipeline-insecure="{{ .PipelineURLInsecure }}" \
--pipeline-token="{{ .PipelineToken }}" \
--pipeline-org-id={{ .OrgID }} \
--pipeline-cluster-id={{ .ClusterID}} \
--pipeline-nodepool={{ .NodePoolName }} \
--taints={{ .Taints }} \
--kubernetes-cloud-provider=vsphere \
--kubernetes-api-server={{ .PublicAddress }}:6443 \
--kubernetes-infrastructure-cidr=$PRIVATE_IP/32 \
--kubernetes-version={{ .KubernetesVersion }} \
--kubernetes-pod-network-cidr="" \
${PKE_EXTRA_ARGS:-}`
