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
	"time"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/secret"
)

type AzurePkeCluster struct {
	model   pke.PKEOnAzureCluster
	secrets SecretStore
	store   pke.AzurePKEClusterStore
}

type SecretStore interface {
	Get(organizationID uint, secretID string) (*secret.SecretItemResponse, error)
	GetByName(organizationID uint, secretName string) (*secret.SecretItemResponse, error)
}

type CommonClusterGetter struct {
	secrets SecretStore
	store   pke.AzurePKEClusterStore
}

func MakeCommonClusterGetter(secrets SecretStore, store pke.AzurePKEClusterStore) CommonClusterGetter {
	return CommonClusterGetter{
		secrets: secrets,
		store:   store,
	}
}

func (g CommonClusterGetter) GetByID(clusterID uint) (*AzurePkeCluster, error) {
	model, err := g.store.GetByID(clusterID)
	if err != nil {
		return nil, err
	}

	cluster := AzurePkeCluster{
		model:   model,
		secrets: g.secrets,
		store:   g.store,
	}

	return &cluster, nil
}

func (a *AzurePkeCluster) GetID() uint {
	return a.model.ID
}

func (a *AzurePkeCluster) GetUID() string {
	return a.model.UID
}

func (a *AzurePkeCluster) GetOrganizationId() uint {
	return a.model.OrganizationID
}

func (a *AzurePkeCluster) GetName() string {
	return a.model.Name
}

func (a *AzurePkeCluster) GetResourceGroupName() string {
	return a.model.ResourceGroup.Name
}

func (a *AzurePkeCluster) GetCloud() string {
	return pkgCluster.Azure
}

func (a *AzurePkeCluster) GetDistribution() string {
	return pkgCluster.PKE
}

func (a *AzurePkeCluster) GetLocation() string {
	return a.model.Location
}

func (a *AzurePkeCluster) GetSecretId() string {
	return a.model.SecretID
}

func (a *AzurePkeCluster) GetSshSecretId() string {
	return a.model.SSHSecretID
}

func (a *AzurePkeCluster) SaveSshSecretId(string) error {
	return errors.New("AzurePkeCluster.SaveSshSecretId is not implemented")
}

func (a *AzurePkeCluster) SaveConfigSecretId(secretID string) error {
	a.model.K8sSecretID = secretID
	return a.store.SetConfigSecretID(a.model.ID, secretID)
}

func (a *AzurePkeCluster) GetConfigSecretId() string {
	return a.model.K8sSecretID
}

func (a *AzurePkeCluster) GetSecretWithValidation() (*secret.SecretItemResponse, error) {
	return a.secrets.Get(a.model.OrganizationID, a.model.SecretID)
}

func (a *AzurePkeCluster) Persist() error {
	return errors.New("AzurePkeCluster.Persist is not implemented")
}

func (a *AzurePkeCluster) DeleteFromDatabase() error {
	return errors.New("AzurePkeCluster.DeleteFromDatabase is not implemented")
}

func (a *AzurePkeCluster) CreateCluster() error {
	return errors.New("AzurePkeCluster.CreateCluster is not implemented")
}

func (a *AzurePkeCluster) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {
	return errors.New("AzurePkeCluster.ValidateCreationFields is not implemented")
}

func (a *AzurePkeCluster) UpdateCluster(*pkgCluster.UpdateClusterRequest, uint) error {
	return errors.New("AzurePkeCluster.UpdateCluster is not implemented")
}

func (a *AzurePkeCluster) UpdateNodePools(*pkgCluster.UpdateNodePoolsRequest, uint) error {
	return errors.New("AzurePkeCluster.UpdateNodePools is not implemented")
}

func (a *AzurePkeCluster) CheckEqualityToUpdate(*pkgCluster.UpdateClusterRequest) error {
	return errors.New("AzurePkeCluster.CheckEqualityToUpdate is not implemented")
}

func (a *AzurePkeCluster) AddDefaultsToUpdate(*pkgCluster.UpdateClusterRequest) {
}

func (a *AzurePkeCluster) DeleteCluster() error {
	return errors.New("AzurePkeCluster.DeleteCluster is not implemented")
}

func (a *AzurePkeCluster) GetScaleOptions() *pkgCluster.ScaleOptions {
	return nil
}

func (a *AzurePkeCluster) SetScaleOptions(*pkgCluster.ScaleOptions) {
}

func (a *AzurePkeCluster) GetTTL() time.Duration {
	return time.Duration(a.model.TtlMinutes) * time.Minute
}

func (a *AzurePkeCluster) SetTTL(t time.Duration) {
	a.model.TtlMinutes = uint(t.Minutes())
	// TODO: persist
}

func (a *AzurePkeCluster) GetAPIEndpoint() (string, error) {
	config, err := a.GetK8sConfig()
	if err != nil {
		return "", errors.WrapIf(err, "failed to get cluster's Kubeconfig")
	}

	return pkgCluster.GetAPIEndpointFromKubeconfig(config)
}

func (a *AzurePkeCluster) GetK8sIpv4Cidrs() (*pkgCluster.Ipv4Cidrs, error) {
	return &pkgCluster.Ipv4Cidrs{
		ServiceClusterIPRanges: []string{"10.10.0.0/16"},
		PodIPRanges:            []string{"10.20.0.0/16"},
	}, nil
	// TODO: use model values once stored/used
	/*
		return &pkgCluster.Ipv4Cidrs{
			ServiceClusterIPRanges: []string{a.model.Kubernetes.Network.ServiceCIDR},
			PodIPRanges:            []string{a.model.Kubernetes.Network.PodCIDR},
		}, nil
	*/
}

func (a *AzurePkeCluster) GetK8sConfig() ([]byte, error) {
	if a.model.K8sSecretID == "" {
		return nil, errors.New("there is no K8s config for the cluster")
	}
	configSecret, err := a.secrets.Get(a.model.OrganizationID, a.model.K8sSecretID)
	if err != nil {
		return nil, errors.Wrap(err, "can't get config from Vault")
	}
	configStr, err := base64.StdEncoding.DecodeString(configSecret.GetValue(secrettype.K8SConfig))
	if err != nil {
		return nil, errors.Wrap(err, "can't decode Kubernetes config")
	}
	return []byte(configStr), nil
}

func (a *AzurePkeCluster) RequiresSshPublicKey() bool {
	return true
}

func (a *AzurePkeCluster) RbacEnabled() bool {
	return a.model.Kubernetes.RBAC
}

func (a *AzurePkeCluster) NeedAdminRights() bool {
	return false
}

func (a *AzurePkeCluster) GetKubernetesUserName() (string, error) {
	return "", errors.New("AzurePkeCluster.GetKubernetesUserName is not implemented")
}

func (a *AzurePkeCluster) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {
	nodePools := make(map[string]*pkgCluster.NodePoolStatus)
	for _, np := range a.model.NodePools {
		nodePools[np.Name] = &pkgCluster.NodePoolStatus{
			Autoscaling:  np.Autoscaling,
			Count:        int(np.DesiredCount),
			InstanceType: np.InstanceType,
			MinCount:     int(np.Min),
			MaxCount:     int(np.Max),
		}
	}

	return &pkgCluster.GetClusterStatusResponse{
		Status:        a.model.Status,
		StatusMessage: a.model.StatusMessage,
		Name:          a.model.Name,
		Location:      a.model.Location,
		Region:        a.model.VirtualNetwork.Location,
		Cloud:         a.GetCloud(),
		Distribution:  a.GetDistribution(),
		ResourceID:    a.model.ID,
		Version:       a.model.Kubernetes.Version,
		NodePools:     nodePools,
		CreatorBaseFields: pkgCommon.CreatorBaseFields{
			CreatedAt:   a.model.CreationTime,
			CreatorName: auth.GetUserNickNameById(a.model.CreatedBy),
			CreatorId:   a.model.CreatedBy,
		}}, nil
}

func (a *AzurePkeCluster) IsReady() (bool, error) {
	return true, nil
}

func (a *AzurePkeCluster) ListNodeNames() (nodeNames pkgCommon.NodeNames, err error) {
	// nodes are labeled in create request
	return
}

func (a *AzurePkeCluster) NodePoolExists(nodePoolName string) bool {
	for _, np := range a.model.NodePools {
		if np.Name == nodePoolName {
			return true
		}
	}
	return false
}

func (a *AzurePkeCluster) SetStatus(status string, statusMessage string) error {
	return a.store.SetStatus(a.model.ID, status, statusMessage)
}

// non-commoncluster methods

// HasK8sConfig returns true if the cluster's k8s config is available
func (a *AzurePkeCluster) HasK8sConfig() (bool, error) {
	config, err := a.GetK8sConfig()
	return len(config) > 0, err
}

func (a *AzurePkeCluster) IsMasterReady() (bool, error) {
	return a.HasK8sConfig()
}

func (a *AzurePkeCluster) GetCurrentWorkflowID() string {
	return a.model.ActiveWorkflowID
}

func (a *AzurePkeCluster) GetCAHash() (string, error) {
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

func (a *AzurePkeCluster) GetPKEOnAzureCluster() pke.PKEOnAzureCluster {
	return a.model
}
