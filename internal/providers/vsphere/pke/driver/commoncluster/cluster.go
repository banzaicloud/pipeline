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
package commoncluster

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/providers/vsphere/pke"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/secret"
)

type VspherePkeCluster struct {
	model   pke.PKEOnVsphereCluster
	secrets SecretStore
	store   pke.ClusterStore
}

type SecretStore interface {
	Get(organizationID uint, secretID string) (*secret.SecretItemResponse, error)
	GetByName(organizationID uint, secretName string) (*secret.SecretItemResponse, error)
}

type CommonClusterGetter struct {
	secrets SecretStore
	store   pke.ClusterStore
}

func MakeCommonClusterGetter(secrets SecretStore, store pke.ClusterStore) CommonClusterGetter {
	return CommonClusterGetter{
		secrets: secrets,
		store:   store,
	}
}

func (g CommonClusterGetter) GetByID(clusterID uint) (*VspherePkeCluster, error) {
	model, err := g.store.GetByID(clusterID)
	if err != nil {
		return nil, err
	}

	cluster := VspherePkeCluster{
		model:   model,
		secrets: g.secrets,
		store:   g.store,
	}

	return &cluster, nil
}

func (a *VspherePkeCluster) GetID() uint {
	return a.model.ID
}

func (a *VspherePkeCluster) GetUID() string {
	return a.model.UID
}

func (a *VspherePkeCluster) GetOrganizationId() uint {
	return a.model.OrganizationID
}

func (a *VspherePkeCluster) GetName() string {
	return a.model.Name
}

func (a *VspherePkeCluster) GetCloud() string {
	return pkgCluster.Vsphere
}

func (a *VspherePkeCluster) GetDistribution() string {
	return pkgCluster.PKE
}

func (a *VspherePkeCluster) GetLocation() string {
	return "n/a"
}

func (a *VspherePkeCluster) GetCreatedBy() uint {
	return a.model.CreatedBy
}

func (a *VspherePkeCluster) GetSecretId() string {
	return a.model.SecretID
}

func (a *VspherePkeCluster) GetSshSecretId() string {
	return a.model.SSHSecretID
}

func (a *VspherePkeCluster) SaveSshSecretId(string) error {
	return errors.New("VspherePkeCluster.SaveSshSecretId is not implemented")
}

func (a *VspherePkeCluster) SaveConfigSecretId(secretID string) error {
	a.model.K8sSecretID = secretID
	return a.store.SetConfigSecretID(a.model.ID, secretID)
}

func (a *VspherePkeCluster) GetConfigSecretId() string {
	return a.model.K8sSecretID
}

func (a *VspherePkeCluster) GetSecretWithValidation() (*secret.SecretItemResponse, error) {
	return a.secrets.Get(a.model.OrganizationID, a.model.SecretID)
}

func (a *VspherePkeCluster) Persist() error {
	return errors.New("VspherePkeCluster.Persist is not implemented")
}

func (a *VspherePkeCluster) DeleteFromDatabase() error {
	return errors.New("VspherePkeCluster.DeleteFromDatabase is not implemented")
}

func (a *VspherePkeCluster) CreateCluster() error {
	return errors.New("VspherePkeCluster.CreateCluster is not implemented")
}

func (a *VspherePkeCluster) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {
	return errors.New("VspherePkeCluster.ValidateCreationFields is not implemented")
}

func (a *VspherePkeCluster) UpdateCluster(*pkgCluster.UpdateClusterRequest, uint) error {
	return errors.New("VspherePkeCluster.UpdateCluster is not implemented")
}

func (a *VspherePkeCluster) UpdateNodePools(*pkgCluster.UpdateNodePoolsRequest, uint) error {
	return errors.New("VspherePkeCluster.UpdateNodePools is not implemented")
}

func (a *VspherePkeCluster) CheckEqualityToUpdate(*pkgCluster.UpdateClusterRequest) error {
	return errors.New("VspherePkeCluster.CheckEqualityToUpdate is not implemented")
}

func (a *VspherePkeCluster) AddDefaultsToUpdate(*pkgCluster.UpdateClusterRequest) {
}

func (a *VspherePkeCluster) DeleteCluster() error {
	return errors.New("VspherePkeCluster.DeleteCluster is not implemented")
}

func (a *VspherePkeCluster) GetAPIEndpoint() (string, error) {
	config, err := a.GetK8sConfig()
	if err != nil {
		return "", errors.WrapIf(err, "failed to get cluster's Kubeconfig")
	}

	return pkgCluster.GetAPIEndpointFromKubeconfig(config)
}

func (a *VspherePkeCluster) GetK8sConfig() ([]byte, error) {
	if a.model.K8sSecretID == "" {
		return nil, errors.New("there is no K8s config for the cluster")
	}
	configSecret, err := a.secrets.Get(a.model.OrganizationID, a.model.K8sSecretID)
	if err != nil {
		return nil, errors.Wrap(err, "can't get config from Vault")
	}
	configStr, err := base64.StdEncoding.DecodeString(configSecret.Values[secrettype.K8SConfig])
	if err != nil {
		return nil, errors.Wrap(err, "can't decode Kubernetes config")
	}
	return configStr, nil
}

func (a *VspherePkeCluster) GetK8sUserConfig() ([]byte, error) {
	return a.GetK8sConfig()
}

func (a *VspherePkeCluster) RequiresSshPublicKey() bool {
	return true
}

func (a *VspherePkeCluster) RbacEnabled() bool {
	return a.model.Kubernetes.RBAC
}

func (a *VspherePkeCluster) NeedAdminRights() bool {
	return false
}

func (a *VspherePkeCluster) GetKubernetesUserName() (string, error) {
	return "", errors.New("VspherePkeCluster.GetKubernetesUserName is not implemented")
}

func (a *VspherePkeCluster) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {
	nodePools := make(map[string]*pkgCluster.NodePoolStatus)
	for _, np := range a.model.NodePools {
		nodePools[np.Name] = &pkgCluster.NodePoolStatus{
			Count:        np.Size,
			InstanceType: np.InstanceType(),
			Vcpu:         np.VCPU,
			Ram:          np.RAM,
			Template:     np.TemplateName,
		}
	}

	return &pkgCluster.GetClusterStatusResponse{
		Status:        a.model.Status,
		StatusMessage: a.model.StatusMessage,
		Name:          a.model.Name,
		Location:      a.GetLocation(),
		Region:        a.GetLocation(),
		Cloud:         a.GetCloud(),
		Distribution:  a.GetDistribution(),
		ResourceID:    a.model.ID,
		Version:       a.model.Kubernetes.Version,
		OIDCEnabled:   a.model.Kubernetes.OIDC.Enabled,
		NodePools:     nodePools,
		CreatorBaseFields: pkgCommon.CreatorBaseFields{
			CreatedAt:   a.model.CreationTime,
			CreatorName: auth.GetUserNickNameById(a.model.CreatedBy),
			CreatorId:   a.model.CreatedBy,
		},
	}, nil
}

func (a *VspherePkeCluster) IsReady() (bool, error) {
	if a.model.SecretID == "" {
		return false, nil
	}
	return true, nil
}

func (a *VspherePkeCluster) NodePoolExists(nodePoolName string) bool {
	for _, np := range a.model.NodePools {
		if np.Name == nodePoolName {
			return true
		}
	}
	return false
}

func (a *VspherePkeCluster) SetStatus(status string, statusMessage string) error {
	return a.store.SetStatus(a.model.ID, status, statusMessage)
}

// non-commoncluster methods

// HasK8sConfig returns true if the cluster's k8s config is available
func (a *VspherePkeCluster) HasK8sConfig() (bool, error) {
	config, err := a.GetK8sConfig()
	return len(config) > 0, err
}

func (a *VspherePkeCluster) IsMasterReady() (bool, error) {
	return a.HasK8sConfig()
}

func (a *VspherePkeCluster) GetCurrentWorkflowID() string {
	return a.model.ActiveWorkflowID
}

func (a *VspherePkeCluster) GetCAHash() (string, error) {
	secret, err := a.secrets.GetByName(a.GetOrganizationId(), fmt.Sprintf("cluster-%d-ca", a.GetID()))
	if err != nil {
		return "", err
	}
	crt := secret.Values[secrettype.KubernetesCACert]
	block, _ := pem.Decode([]byte(crt))
	if block == nil {
		return "", errors.New("failed to parse certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", errors.WrapIff(err, "failed to parse certificate")
	}
	h := sha256.Sum256(cert.RawSubjectPublicKeyInfo)
	return fmt.Sprintf("sha256:%s", hex.EncodeToString(h[:])), nil
}

func (a *VspherePkeCluster) GetPKEOnVsphereCluster() pke.PKEOnVsphereCluster {
	return a.model
}
