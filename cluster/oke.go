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
	"fmt"

	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	oracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/cluster"
	oracleClusterManager "github.com/banzaicloud/pipeline/pkg/providers/oracle/cluster/manager"
	modelOracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/model"
	"github.com/banzaicloud/pipeline/pkg/providers/oracle/network"
	"github.com/banzaicloud/pipeline/pkg/providers/oracle/oci"
	secretOracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/secret"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/pkg/errors"
)

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
func CreateOKEClusterFromRequest(request *pkgCluster.CreateClusterRequest, orgId, userId uint) (*OKECluster, error) {

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
		return errors.Wrap(err, "could not get ssh pubkey")
	}

	return cm.ManageOKECluster(&o.modelCluster.OKE)
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
		return errors.Wrap(err, "could not get ssh pubkey")
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

//Persist save the cluster model
func (o *OKECluster) Persist(status, statusMessage string) error {

	return o.modelCluster.UpdateStatus(status, statusMessage)
}

// DownloadK8sConfig downloads the kubeconfig file from cloud
func (o *OKECluster) DownloadK8sConfig() ([]byte, error) {

	oci, err := o.GetOCIWithRegion(o.modelCluster.Location)
	if err != nil {
		return nil, err
	}

	ce, err := oci.NewContainerEngineClient()
	if err != nil {
		return nil, err
	}

	return ce.GetK8SConfig(o.modelCluster.OKE.OCID)
}

//GetName returns the name of the cluster
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

//GetStatus gets cluster status
func (o *OKECluster) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {

	nodePools := make(map[string]*pkgCluster.NodePoolStatus)
	for _, np := range o.modelCluster.OKE.NodePools {
		if np != nil {
			count := getNodeCount(np)
			nodePools[np.Name] = &pkgCluster.NodePoolStatus{
				Count:        count,
				Autoscaling:  false,
				MinCount:     count,
				MaxCount:     count,
				InstanceType: np.Shape,
				Image:        np.Image,
				Version:      np.Version,
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
		Logging:           o.GetLogging(),
		Monitoring:        o.GetMonitoring(),
		SecurityScan:      o.GetSecurityScan(),
		CreatorBaseFields: *NewCreatorBaseFields(o.modelCluster.CreatedAt, o.modelCluster.CreatedBy),
		NodePools:         nodePools,
		Region:            o.modelCluster.Location,
	}, nil
}

func getNodeCount(np *modelOracle.NodePool) int {
	return int(np.QuantityPerSubnet) * len(np.Subnets)
}

//GetID returns the specified cluster id
func (o *OKECluster) GetID() uint {
	return o.modelCluster.ID
}

func (o *OKECluster) GetUID() string {
	return o.modelCluster.UID
}

//GetModel returns the whole clusterModel
func (o *OKECluster) GetModel() *model.ClusterModel {
	return o.modelCluster
}

//CheckEqualityToUpdate validates the update request
func (o *OKECluster) CheckEqualityToUpdate(r *pkgCluster.UpdateClusterRequest) error {

	cluster := o.modelCluster.OKE.GetClusterRequestFromModel()

	log.Info("Check stored & updated cluster equals")

	return isDifferent(r.OKE, cluster)
}

//AddDefaultsToUpdate adds defaults to update request
func (o *OKECluster) AddDefaultsToUpdate(r *pkgCluster.UpdateClusterRequest) {

	r.UpdateProperties.OKE.AddDefaults()
}

//GetAPIEndpoint returns the Kubernetes Api endpoint
func (o *OKECluster) GetAPIEndpoint() (string, error) {

	oci, err := o.GetOCIWithRegion(o.modelCluster.Location)
	if err != nil {
		return o.APIEndpoint, err
	}

	ce, err := oci.NewContainerEngineClient()
	if err != nil {
		return o.APIEndpoint, err
	}

	cluster, err := ce.GetClusterByID(&o.modelCluster.OKE.OCID)
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

//GetSecretId retrieves the secret id
func (o *OKECluster) GetSecretId() string {
	return o.modelCluster.SecretId
}

// RequiresSshPublicKey returns true if an ssh public key is needed for the cluster for bootstrapping it.
func (o *OKECluster) RequiresSshPublicKey() bool {
	return true
}

//GetSshSecretId retrieves the ssh secret id
func (o *OKECluster) GetSshSecretId() string {
	return o.modelCluster.SshSecretId
}

// SaveSshSecretId saves the ssh secret id to database
func (o *OKECluster) SaveSshSecretId(sshSecretId string) error {
	return o.modelCluster.UpdateSshSecret(sshSecretId)
}

// UpdateStatus updates cluster status in database
func (o *OKECluster) UpdateStatus(status, statusMessage string) error {
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

// GetClusterDetails gets cluster details from cloud
func (o *OKECluster) GetClusterDetails() (*pkgCluster.DetailsResponse, error) {
	ready, err := o.IsReady()
	if err != nil {
		return nil, err
	}

	if !ready {
		return nil, pkgErrors.ErrorClusterNotReady
	}

	status, err := o.GetStatus()
	if err != nil {
		return nil, err
	}

	nodePools := make(map[string]*pkgCluster.NodePoolDetails)
	for _, np := range o.modelCluster.OKE.NodePools {
		if np != nil {
			nodePools[np.Name] = &pkgCluster.NodePoolDetails{
				CreatorBaseFields: *NewCreatorBaseFields(np.CreatedAt, np.CreatedBy),
				NodePoolStatus:    *status.NodePools[np.Name],
			}
		}
	}

	// todo needs to add other fields
	return &pkgCluster.DetailsResponse{
		Id:                       status.ResourceID,
		MasterVersion:            o.modelCluster.OKE.Version,
		NodePools:                nodePools,
		GetClusterStatusResponse: *status,
	}, nil
}

// IsReady checks if the cluster is running according to the cloud provider.
func (o *OKECluster) IsReady() (bool, error) {

	oci, err := o.GetOCIWithRegion(o.modelCluster.Location)
	if err != nil {
		return false, err
	}

	ce, err := oci.NewContainerEngineClient()
	if err != nil {
		return false, err
	}

	cluster, err := ce.GetClusterByID(&o.modelCluster.OKE.OCID)
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
			err = errors.Wrap(deleteError, err.Error())
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

// GetClusterManager creates a new oracleClusterManager.ClusterManager
func (o *OKECluster) GetClusterManager() (manager *oracleClusterManager.ClusterManager, err error) {

	oci, err := o.GetOCIWithRegion(o.modelCluster.Location)
	if err != nil {
		return manager, err
	}

	return oracleClusterManager.NewClusterManager(oci), nil
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
		return VCNID, fmt.Errorf("Invalid VCN!")
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
	if len(networkValues.LBSubnetIDs) != 2 {
		return r, fmt.Errorf("Invalid network config: there must be 2 loadbalancer subnets!")
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

// ListNodeNames returns node names to label them
func (o *OKECluster) ListNodeNames() (nodeNames pkgCommon.NodeNames, err error) {
	// nodes are labeled in create request
	return
}

// RbacEnabled returns true if rbac enabled on the cluster
func (o *OKECluster) RbacEnabled() bool {
	return true
}

// SecurityScan returns true if security scan enabled on the cluster
func (o *OKECluster) GetSecurityScan() bool {
	return o.modelCluster.SecurityScan
}

// SetSecurityScan returns true if security scan enabled on the cluster
func (o *OKECluster) SetSecurityScan(scan bool) {
	o.modelCluster.SecurityScan = scan
}

// GetLogging returns true if logging enabled on the cluster
func (o *OKECluster) GetLogging() bool {
	return o.modelCluster.Logging
}

// SetLogging returns true if logging enabled on the cluster
func (o *OKECluster) SetLogging(l bool) {
	o.modelCluster.Logging = l
}

// GetMonitoring returns true if momnitoring enabled on the cluster
func (o *OKECluster) GetMonitoring() bool {
	return o.modelCluster.Monitoring
}

// SetMonitoring returns true if monitoring enabled on the cluster
func (o *OKECluster) SetMonitoring(l bool) {
	o.modelCluster.Monitoring = l
}

// NeedAdminRights returns true if rbac is enabled and need to create a cluster role binding to user
func (o *OKECluster) NeedAdminRights() bool {
	return true
}

// GetKubernetesUserName returns the user ID which needed to create a cluster role binding which gives admin rights to the user
func (o *OKECluster) GetKubernetesUserName() (string, error) {

	s, err := o.GetSecretWithValidation()
	if err != nil {
		return "", errors.Wrap(err, "error getting secret")
	}

	if s.Values[pkgSecret.OracleUserOCID] == "" {
		return "", errors.New("empty user OCID")
	}

	return s.Values[pkgSecret.OracleUserOCID], nil

}

// GetCreatedBy returns cluster create userID.
func (o *OKECluster) GetCreatedBy() uint {
	return o.modelCluster.CreatedBy
}

func (o *OKECluster) getSSHPubKey() (string, error) {

	sshSecret, err := o.getSshSecret(o)
	if err != nil {
		return "", err
	}

	sshKey := secret.NewSSHKeyPair(sshSecret)

	return sshKey.PublicKeyData, nil
}
