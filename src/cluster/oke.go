// Copyright © 2018 Banzai Cloud
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

package cluster

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"emperror.dev/errors"
	"github.com/ghodss/yaml"
	ociCommon "github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/containerengine"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/internal/secret/ssh/sshadapter"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	oracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/cluster"
	oracleClusterManager "github.com/banzaicloud/pipeline/pkg/providers/oracle/cluster/manager"
	modelOracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/model"
	"github.com/banzaicloud/pipeline/pkg/providers/oracle/network"
	"github.com/banzaicloud/pipeline/pkg/providers/oracle/oci"
	secretOracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/secret"
	"github.com/banzaicloud/pipeline/src/model"
	"github.com/banzaicloud/pipeline/src/secret"
)

const kubeconfigTokenURL = "https://containerengine.%s.oraclecloud.com/cluster_request/%s"

// OKECluster struct for OKE cluster
type OKECluster struct {
	modelCluster *model.ClusterModel
	APIEndpoint  string
	CommonClusterBase
}

// CreateOKEClusterFromModel creates ClusterModel struct from model
func CreateOKEClusterFromModel(clusterModel *model.ClusterModel) (*OKECluster, error) {
	okeCluster := OKECluster{
		modelCluster: clusterModel,
	}
	return &okeCluster, nil
}

// CreateOKEClusterFromRequest creates ClusterModel struct from the request
func CreateOKEClusterFromRequest(request *pkgCluster.CreateClusterRequest, orgId uint, userId uint) (*OKECluster, error) {

	var oke OKECluster

	oke.modelCluster = &model.ClusterModel{
		Name:           request.Name,
		Location:       request.Location,
		Cloud:          request.Cloud,
		OrganizationId: orgId,
		SecretId:       request.SecretId,
		CreatedBy:      userId,
		Distribution:   pkgCluster.OKE,
	}
	updateScaleOptions(&oke.modelCluster.ScaleOptions, request.ScaleOptions)

	VCNID, err := oke.CreatePreconfiguredVCN(request.Name)
	if err != nil {
		return &oke, err
	}

	properties, err := oke.PopulateNetworkValues(request.Properties.CreateClusterOKE, VCNID)
	if err != nil {
		return &oke, err
	}
	request.Properties.CreateClusterOKE = properties

	Model, err := modelOracle.CreateModelFromCreateRequest(request, userId)
	if err != nil {
		return &oke, err
	}

	oke.modelCluster.OKE = Model

	return &oke, nil
}

// CreateCluster creates a new cluster
func (o *OKECluster) CreateCluster() error {

	log.Info("Start creating Oracle cluster")

	cm, err := o.GetClusterManager()
	if err != nil {
		return err
	}

	o.modelCluster.OKE.SSHPubKey, err = o.getSSHPubKey()
	if err != nil {
		return errors.WrapIf(err, "could not get ssh pubkey")
	}

	return cm.ManageOKECluster(&o.modelCluster.OKE)
}

// UpdateNodePools updates nodes pools of a cluster
func (o *OKECluster) UpdateNodePools(request *pkgCluster.UpdateNodePoolsRequest, userId uint) error {
	return nil
}

// UpdateCluster updates the cluster
func (o *OKECluster) UpdateCluster(r *pkgCluster.UpdateClusterRequest, userId uint) error {

	updated, err := o.PopulateNetworkValues(r.UpdateProperties.OKE, o.modelCluster.OKE.VCNID)
	if err != nil {
		return err
	}
	r.UpdateProperties.OKE = updated

	model, err := modelOracle.CreateModelFromUpdateRequest(o.modelCluster.OKE, r, userId)
	if err != nil {
		return err
	}

	model.SSHPubKey, err = o.getSSHPubKey()
	if err != nil {
		return errors.WrapIf(err, "could not get ssh pubkey")
	}

	o.modelCluster.OKE = model

	cm, err := o.GetClusterManager()
	if err != nil {
		return err
	}

	err = cm.ManageOKECluster(&model)
	if err != nil {
		return err
	}

	// remove node pools from model which are marked for deleting
	nodePools := make([]*modelOracle.NodePool, 0)
	for _, np := range model.NodePools {
		if !np.Delete {
			nodePools = append(nodePools, np)
		}
	}

	model.NodePools = nodePools
	o.modelCluster.OKE = model

	return err
}

// DeleteCluster deletes cluster
func (o *OKECluster) DeleteCluster() error {

	// mark cluster model to deleting
	o.modelCluster.OKE.Delete = true

	cm, err := o.GetClusterManager()
	if err != nil {
		return err
	}

	err = cm.ManageOKECluster(&o.modelCluster.OKE)
	if err != nil {
		return err
	}

	err = o.DeletePreconfiguredVCN(o.modelCluster.OKE.VCNID)
	if err != nil {
		return err
	}

	return nil
}

// Persist save the cluster model
// Deprecated: Do not use.
func (o *OKECluster) Persist() error {
	return errors.WrapIf(o.modelCluster.Save(), "failed to persist cluster")
}

// DownloadK8sConfig downloads the kubeconfig file from cloud
func (o *OKECluster) DownloadK8sConfig() ([]byte, error) {

	oci, err := o.GetOCIWithRegion(o.modelCluster.Location)
	if err != nil {
		return nil, errors.WrapIf(err, "could not get OCI client")
	}

	ce, err := oci.NewContainerEngineClient()
	if err != nil {
		return nil, errors.WrapIf(err, "could not get container engine client")
	}

	k8sConfig, err := ce.GetK8SConfig(o.modelCluster.OKE.OCID)
	if err != nil {
		return nil, errors.WrapIf(err, "could not retrieve kubeconfig from provider")
	}

	k8sConfig, err = o.replaceKubeconfigExecAuthWithSAToken(oci, k8sConfig)
	if err != nil {
		return nil, errors.WrapIf(err, "could not replace exec auth in kubeconfig")
	}

	return k8sConfig, nil
}

func (o *OKECluster) replaceKubeconfigExecAuthWithSAToken(oci *oci.OCI, k8sConfig []byte) ([]byte, error) {
	var config clientcmdv1.Config
	err := yaml.Unmarshal(k8sConfig, &config)
	if err != nil {
		return nil, errors.WrapIf(err, "could not unmarshal kubeconfig")
	}

	execFound := false
	for _, ai := range config.AuthInfos {
		if ai.AuthInfo.Exec != nil {
			execFound = true
			break
		}
	}

	if !execFound {
		return k8sConfig, nil
	}

	signer := ociCommon.DefaultRequestSigner(oci.GetConfig())
	region, err := oci.GetConfig().Region()
	if err != nil {
		return nil, errors.WrapIf(err, "could not get region")
	}

	u, err := url.Parse(fmt.Sprintf(kubeconfigTokenURL, region, o.modelCluster.OKE.OCID))
	if err != nil {
		return nil, errors.WrapIf(err, "could not parse token request URL")
	}

	req := &http.Request{
		Method: http.MethodGet,
		URL:    u,
		Header: make(http.Header),
	}
	req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	err = signer.Sign(req)
	if err != nil {
		return nil, errors.WrapIf(err, "could not create signature for token request")
	}

	q := u.Query()
	q.Add("authorization", req.Header.Get("Authorization"))
	q.Add("date", req.Header.Get("Date"))
	u.RawQuery = q.Encode()

	token := base64.URLEncoding.EncodeToString([]byte(u.String()))

	for i, ai := range config.AuthInfos {
		if ai.AuthInfo.Exec != nil {
			ai.AuthInfo.Exec = nil
			ai.AuthInfo.Token = token
		}
		config.AuthInfos[i] = ai
	}

	k8sConfig, err = yaml.Marshal(config)
	if err != nil {
		return nil, errors.WrapIf(err, "could not marshal kubeconfig")
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(k8sConfig)
	if err != nil {
		return nil, errors.WrapIf(err, "could not get rest config from yaml")
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.WrapIf(err, "could not create kubernetes clientset")
	}

	saToken, err := generateServiceAccountToken(clientset)
	if err != nil {
		return nil, errors.WrapIf(err, "could not create service account for pipeline")
	}

	for i, ai := range config.AuthInfos {
		if ai.AuthInfo.Token != "" {
			ai.AuthInfo.Token = saToken
		}
		config.AuthInfos[i] = ai
	}

	k8sConfig, err = yaml.Marshal(config)
	if err != nil {
		return nil, errors.WrapIf(err, "could not marshal kubeconfig")
	}

	return k8sConfig, nil
}

// GetName returns the name of the cluster
func (o *OKECluster) GetName() string {
	return o.modelCluster.Name
}

// GetCloud returns the cloud type of the cluster
func (o *OKECluster) GetCloud() string {
	return pkgCluster.Oracle
}

// GetDistribution returns the distribution type of the cluster
func (o *OKECluster) GetDistribution() string {
	return o.modelCluster.Distribution
}

// GetStatus gets cluster status
func (o *OKECluster) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {

	nodePools := make(map[string]*pkgCluster.NodePoolStatus)
	for _, np := range o.modelCluster.OKE.NodePools {
		if np != nil {
			count := getNodeCount(np)
			nodePools[np.Name] = &pkgCluster.NodePoolStatus{
				Count:             count,
				Autoscaling:       false,
				MinCount:          count,
				MaxCount:          count,
				InstanceType:      np.Shape,
				Image:             np.Image,
				Version:           np.Version,
				CreatorBaseFields: *NewCreatorBaseFields(np.CreatedAt, np.CreatedBy),
				Labels:            np.Labels,
			}
		}
	}

	return &pkgCluster.GetClusterStatusResponse{
		Status:            o.modelCluster.Status,
		StatusMessage:     o.modelCluster.StatusMessage,
		Name:              o.modelCluster.Name,
		Location:          o.modelCluster.Location,
		Cloud:             pkgCluster.Oracle,
		Distribution:      o.modelCluster.Distribution,
		Version:           o.modelCluster.OKE.Version,
		ResourceID:        o.GetID(),
		CreatorBaseFields: *NewCreatorBaseFields(o.modelCluster.CreatedAt, o.modelCluster.CreatedBy),
		NodePools:         nodePools,
		Region:            o.modelCluster.Location,
		StartedAt:         o.modelCluster.StartedAt,
	}, nil
}

func getNodeCount(np *modelOracle.NodePool) int {
	return int(np.QuantityPerSubnet) * len(np.Subnets)
}

// GetID returns the specified cluster id
func (o *OKECluster) GetID() uint {
	return o.modelCluster.ID
}

func (o *OKECluster) GetUID() string {
	return o.modelCluster.UID
}

// GetModel returns the whole clusterModel
func (o *OKECluster) GetModel() *model.ClusterModel {
	return o.modelCluster
}

// CheckEqualityToUpdate validates the update request
func (o *OKECluster) CheckEqualityToUpdate(r *pkgCluster.UpdateClusterRequest) error {

	cluster := o.modelCluster.OKE.GetClusterRequestFromModel()

	log.Info("Check stored & updated cluster equals")

	return isDifferent(r.OKE, cluster)
}

// AddDefaultsToUpdate adds defaults to update request
func (o *OKECluster) AddDefaultsToUpdate(r *pkgCluster.UpdateClusterRequest) {

	r.UpdateProperties.OKE.AddDefaults() // nolint: errcheck
}

// GetAPIEndpoint returns the Kubernetes Api endpoint
func (o *OKECluster) GetAPIEndpoint() (string, error) {
	cluster, err := o.GetCluster()
	if err != nil {
		return o.APIEndpoint, err
	}

	o.APIEndpoint = fmt.Sprintf("https://%s", *cluster.Endpoints.Kubernetes)

	return o.APIEndpoint, nil
}

// DeleteFromDatabase deletes model from the database
func (o *OKECluster) DeleteFromDatabase() error {
	err := o.modelCluster.Delete()
	if err != nil {
		return err
	}

	err = o.modelCluster.OKE.Cleanup()
	if err != nil {
		return err
	}

	o.modelCluster = nil
	return nil
}

// GetOrganizationId gets org where the cluster belongs
func (o *OKECluster) GetOrganizationId() uint {
	return o.modelCluster.OrganizationId
}

// GetLocation gets where the cluster is.
func (o *OKECluster) GetLocation() string {
	return o.modelCluster.Location
}

// GetSecretId retrieves the secret id
func (o *OKECluster) GetSecretId() string {
	return o.modelCluster.SecretId
}

// RequiresSshPublicKey returns true if an ssh public key is needed for the cluster for bootstrapping it.
func (o *OKECluster) RequiresSshPublicKey() bool {
	return true
}

// GetSshSecretId retrieves the ssh secret id
func (o *OKECluster) GetSshSecretId() string {
	return o.modelCluster.SshSecretId
}

// SaveSshSecretId saves the ssh secret id to database
func (o *OKECluster) SaveSshSecretId(sshSecretId string) error {
	return o.modelCluster.UpdateSshSecret(sshSecretId)
}

// SetStatus sets the cluster's status
func (o *OKECluster) SetStatus(status, statusMessage string) error {
	return o.modelCluster.UpdateStatus(status, statusMessage)
}

// NodePoolExists returns true if node pool with nodePoolName exists
func (o *OKECluster) NodePoolExists(nodePoolName string) bool {
	for _, np := range o.modelCluster.OKE.NodePools {
		if np != nil && np.Name == nodePoolName {
			return true
		}
	}
	return false
}

// IsReady checks if the cluster is running according to the cloud provider.
func (o *OKECluster) IsReady() (bool, error) {
	cluster, err := o.GetCluster()
	if err != nil {
		return false, err
	}

	return cluster.LifecycleState == "ACTIVE", nil
}

// ValidateCreationFields validates all field
func (o *OKECluster) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {

	cm, err := o.GetClusterManager()
	if err != nil {
		return err
	}

	err = cm.ValidateModel(&o.modelCluster.OKE)
	if err != nil {
		deleteError := o.DeletePreconfiguredVCN(o.modelCluster.OKE.VCNID)
		if deleteError != nil {
			err = errors.WrapIf(deleteError, err.Error())
		}
		return err
	}

	return nil
}

// GetSecretWithValidation returns secret from vault
func (o *OKECluster) GetSecretWithValidation() (*secret.SecretItemResponse, error) {
	return o.CommonClusterBase.getSecret(o)
}

// SaveConfigSecretId saves the config secret id in database
func (o *OKECluster) SaveConfigSecretId(configSecretId string) error {
	return o.modelCluster.UpdateConfigSecret(configSecretId)
}

// GetConfigSecretId return config secret id
func (o *OKECluster) GetConfigSecretId() string {
	return o.modelCluster.ConfigSecretId
}

// GetK8sConfig returns the Kubernetes config
func (o *OKECluster) GetK8sConfig() ([]byte, error) {
	return o.CommonClusterBase.getConfig(o)
}

// GetK8sUserConfig returns the Kubernetes config
func (o *OKECluster) GetK8sUserConfig() ([]byte, error) {
	return o.GetK8sConfig()
}

// GetClusterManager creates a new oracleClusterManager.ClusterManager
func (o *OKECluster) GetClusterManager() (manager *oracleClusterManager.ClusterManager, err error) {

	oci, err := o.GetOCIWithRegion(o.modelCluster.Location)
	if err != nil {
		return manager, err
	}

	return oracleClusterManager.NewClusterManager(oci), nil
}

// GetCluster returns the Kubernetes cluster
func (o *OKECluster) GetCluster() (cluster containerengine.Cluster, err error) {
	oci, err := o.GetOCIWithRegion(o.modelCluster.Location)
	if err != nil {
		return cluster, err
	}

	ce, err := oci.NewContainerEngineClient()
	if err != nil {
		return cluster, err
	}

	cluster, err = ce.GetClusterByID(&o.modelCluster.OKE.OCID)
	if err != nil {
		return cluster, err
	}

	return cluster, nil
}

// GetOCI creates a new oci.OCI
func (o *OKECluster) GetOCI() (OCI *oci.OCI, err error) {

	s, err := o.CommonClusterBase.getSecret(o)
	if err != nil {
		return OCI, err
	}

	OCI, err = oci.NewOCI(secretOracle.CreateOCICredential(s.Values))
	if err != nil {
		return OCI, err
	}

	OCI.SetLogger(log)

	return OCI, err
}

// GetOCIWithRegion creates a new oci.OCI with the given region
func (o *OKECluster) GetOCIWithRegion(region string) (OCI *oci.OCI, err error) {

	OCI, err = o.GetOCI()
	if err != nil {
		return OCI, err
	}

	err = OCI.ChangeRegion(region)

	return OCI, err
}

// CreatePreconfiguredVCN creates a preconfigured VCN with the given name
func (o *OKECluster) CreatePreconfiguredVCN(name string) (VCNID string, err error) {

	oci, err := o.GetOCIWithRegion(o.modelCluster.Location)
	if err != nil {
		return
	}

	m := network.NewVCNManager(oci)
	vcn, err := m.Create(fmt.Sprintf("p-%s", name))
	if err != nil {
		return
	}

	if vcn.Id == nil {
		return VCNID, errors.New("invalid VCN")
	}

	VCNID = *vcn.Id

	return
}

// DeletePreconfiguredVCN deletes a preconfigured VCN by id
func (o *OKECluster) DeletePreconfiguredVCN(VCNID string) (err error) {

	oci, err := o.GetOCIWithRegion(o.modelCluster.Location)
	if err != nil {
		return
	}

	m := network.NewVCNManager(oci)
	return m.Delete(&VCNID)
}

// PopulateNetworkValues fills network related values in the request object
func (o *OKECluster) PopulateNetworkValues(r *oracle.Cluster, VCNID string) (*oracle.Cluster, error) {

	oci, err := o.GetOCIWithRegion(o.modelCluster.Location)
	if err != nil {
		return r, err
	}

	m := network.NewVCNManager(oci)
	networkValues, err := m.GetNetworkValues(VCNID)
	if err != nil {
		return r, err
	}

	r.SetVCNID(VCNID)
	if len(networkValues.LBSubnetIDs) != 2 || networkValues.LBSubnetIDs[0] == networkValues.LBSubnetIDs[1] {
		return r, errors.New("invalid network config: there must be exactly 2 different load balancer subnets specified")
	}
	r.SetLBSubnetID1(networkValues.LBSubnetIDs[0])
	r.SetLBSubnetID2(networkValues.LBSubnetIDs[1])

	for _, np := range r.NodePools {
		quanityPerSubnet, subnetIDs := o.GetPoolQuantityValues(np.Count, networkValues)
		np.SetQuantityPerSubnet(quanityPerSubnet)
		np.SetSubnetIDs(subnetIDs)
	}

	return r, nil
}

// GetPoolQuantityValues calculates quantityPerSubnet and SubnetIDS for the given instance count
func (o *OKECluster) GetPoolQuantityValues(count uint, networkValues network.NetworkValues) (qps uint, subnetIDS []string) {

	if count == 0 || len(networkValues.WNSubnetIDs) < 3 {
		return
	}

	qps = count
	subnetIDS = networkValues.WNSubnetIDs[0:1]
	if count%3 == 0 {
		qps = count / 3
		subnetIDS = networkValues.WNSubnetIDs[0:3]
	} else if count%2 == 0 {
		qps = count / 2
		subnetIDS = networkValues.WNSubnetIDs[0:2]
	}

	return qps, subnetIDS
}

// RbacEnabled returns true if rbac enabled on the cluster
func (o *OKECluster) RbacEnabled() bool {
	return true
}

// getScaleOptionsFromModelV1 returns scale options for the cluster
func (o *OKECluster) GetScaleOptions() *pkgCluster.ScaleOptions {
	return getScaleOptionsFromModel(o.modelCluster.ScaleOptions)
}

// SetScaleOptions sets scale options for the cluster
func (o *OKECluster) SetScaleOptions(scaleOptions *pkgCluster.ScaleOptions) {
	updateScaleOptions(&o.modelCluster.ScaleOptions, scaleOptions)
}

// GetKubernetesUserName returns the user ID which needed to create a cluster role binding which gives admin rights to the user
func (o *OKECluster) GetKubernetesUserName() (string, error) {

	s, err := o.GetSecretWithValidation()
	if err != nil {
		return "", errors.WrapIf(err, "error getting secret")
	}

	if s.Values[secrettype.OracleUserOCID] == "" {
		return "", errors.New("empty user OCID")
	}

	return s.Values[secrettype.OracleUserOCID], nil

}

func (o *OKECluster) getSSHPubKey() (string, error) {

	sshSecret, err := o.getSshSecret(o)
	if err != nil {
		return "", err
	}

	sshKey := sshadapter.KeyPairFromSecret(sshSecret)

	return sshKey.PublicKeyData, nil
}
