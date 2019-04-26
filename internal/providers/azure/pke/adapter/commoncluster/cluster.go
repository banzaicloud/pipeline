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
	"time"

	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke/adapter"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/goph/emperror"
	"github.com/jinzhu/gorm"
)

type AzurePkeCluster struct {
	store pke.AzurePKEClusterStore
	model pke.PKEOnAzureCluster
}

func GetCommonClusterByID(clusterID uint, db *gorm.DB) (*AzurePkeCluster, error) {
	store := adapter.NewGORMAzurePKEClusterStore(db)
	model, err := store.GetByID(clusterID)
	if err != nil {
		return nil, err
	}

	cluster := AzurePkeCluster{
		store: store,
		model: model,
	}

	return &cluster, nil
}

func (a *AzurePkeCluster) GetID() uint {
	return a.model.ID
}

func (a *AzurePkeCluster) GetUID() string {
	panic("not implemented")
}

func (a *AzurePkeCluster) GetOrganizationId() uint {
	return a.model.OrganizationID
}

func (a *AzurePkeCluster) GetName() string {
	return a.model.Name
}

func (a *AzurePkeCluster) GetCloud() string {
	panic("not implemented")
}

func (a *AzurePkeCluster) GetDistribution() string {
	panic("not implemented")
}

func (a *AzurePkeCluster) GetLocation() string {
	panic("not implemented")
}

func (a *AzurePkeCluster) GetCreatedBy() uint {
	panic("not implemented")
}

func (a *AzurePkeCluster) GetSecretId() string {
	panic("not implemented")
}

func (a *AzurePkeCluster) GetSshSecretId() string {
	panic("not implemented")
}

func (a *AzurePkeCluster) SaveSshSecretId(string) error {
	panic("not implemented")
}

func (a *AzurePkeCluster) SaveConfigSecretId(secretID string) error {
	a.model.K8sSecretID = secretID
	return a.store.SetConfigSecretId(a.model.ID, secretID)
}

func (a *AzurePkeCluster) GetConfigSecretId() string {
	return a.model.K8sSecretID
}

func (a *AzurePkeCluster) GetSecretWithValidation() (*secret.SecretItemResponse, error) {
	return secret.Store.Get(a.model.OrganizationID, a.model.SecretID)
}

func (a *AzurePkeCluster) Persist() error {
	panic("not implemented")
}

func (a *AzurePkeCluster) DeleteFromDatabase() error {
	panic("not implemented")
}

func (a *AzurePkeCluster) CreateCluster() error {
	panic("not implemented")
}

func (a *AzurePkeCluster) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {
	panic("not implemented")
}

func (a *AzurePkeCluster) UpdateCluster(*pkgCluster.UpdateClusterRequest, uint) error {
	panic("not implemented")
}

func (a *AzurePkeCluster) UpdateNodePools(*pkgCluster.UpdateNodePoolsRequest, uint) error {
	panic("not implemented")
}

func (a *AzurePkeCluster) CheckEqualityToUpdate(*pkgCluster.UpdateClusterRequest) error {
	panic("not implemented")
}

func (a *AzurePkeCluster) AddDefaultsToUpdate(*pkgCluster.UpdateClusterRequest) {
	panic("not implemented")
}

func (a *AzurePkeCluster) DeleteCluster() error {
	panic("not implemented")
}

func (a *AzurePkeCluster) GetScaleOptions() *pkgCluster.ScaleOptions {
	return nil
}

func (a *AzurePkeCluster) SetScaleOptions(*pkgCluster.ScaleOptions) {
	panic("not implemented")
}

func (a *AzurePkeCluster) GetTTL() time.Duration {
	panic("not implemented")
}

func (a *AzurePkeCluster) SetTTL(time.Duration) {
	panic("not implemented")
}

func (a *AzurePkeCluster) DownloadK8sConfig() ([]byte, error) {
	panic("not implemented")
}

func (c *AzurePkeCluster) GetAPIEndpoint() (string, error) {
	config, err := c.GetK8sConfig()
	if err != nil {
		return "", emperror.Wrap(err, "failed to get cluster's Kubeconfig")
	}

	return pkgCluster.GetAPIEndpointFromKubeconfig(config)
}

func (a *AzurePkeCluster) GetK8sIpv4Cidrs() (*pkgCluster.Ipv4Cidrs, error) {
	panic("not implemented")
}

func (a *AzurePkeCluster) GetK8sConfig() ([]byte, error) {
	panic("not implemented")
}

func (a *AzurePkeCluster) RequiresSshPublicKey() bool {
	panic("not implemented")
}

func (a *AzurePkeCluster) RbacEnabled() bool {
	panic("not implemented")
}

func (a *AzurePkeCluster) NeedAdminRights() bool {
	panic("not implemented")
}

func (a *AzurePkeCluster) GetKubernetesUserName() (string, error) {
	panic("not implemented")
}

func (a *AzurePkeCluster) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {
	return &pkgCluster.GetClusterStatusResponse{}, nil
}

func (a *AzurePkeCluster) IsReady() (bool, error) {
	return true, nil
}

func (a *AzurePkeCluster) ListNodeNames() (pkgCommon.NodeNames, error) {
	panic("not implemented")
}

func (a *AzurePkeCluster) NodePoolExists(nodePoolName string) bool {
	panic("not implemented")
}

func (a *AzurePkeCluster) GetSecurityScan() bool {
	panic("not implemented")
}

func (a *AzurePkeCluster) SetSecurityScan(scan bool) {
	panic("not implemented")
}

func (a *AzurePkeCluster) GetLogging() bool {
	panic("not implemented")
}

func (a *AzurePkeCluster) SetLogging(l bool) {
	panic("not implemented")
}

func (a *AzurePkeCluster) GetMonitoring() bool {
	panic("not implemented")
}

func (a *AzurePkeCluster) SetMonitoring(m bool) {
	panic("not implemented")
}

func (a *AzurePkeCluster) GetServiceMesh() bool {
	panic("not implemented")
}

func (a *AzurePkeCluster) SetServiceMesh(m bool) {
	panic("not implemented")
}

func (a *AzurePkeCluster) SetStatus(status string, statusMessage string) error {
	panic("not implemented")
}

// non-commoncluster methods

// HasK8sConfig returns true if the cluster's k8s config is available
func (a *AzurePkeCluster) HasK8sConfig() (bool, error) {
	panic("todo")
}
