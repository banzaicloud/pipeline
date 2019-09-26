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

package dns

import (
	"context"

	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/secret"
)

type arr = []interface{}
type obj = map[string]interface{}

type dummyClusterGetter struct {
	Clusters map[uint]clusterfeatureadapter.Cluster
}

func (d dummyClusterGetter) GetClusterByIDOnly(ctx context.Context, clusterID uint) (clusterfeatureadapter.Cluster, error) {
	return d.Clusters[clusterID], nil
}

type dummyCluster struct {
	K8sConfig []byte
	Name      string
	OrgID     uint
	ID        uint
	UID       string
	Ready     bool
	NodePools map[string]bool
	Rbac      bool
}

func (d dummyCluster) SetSecurityScan(scan bool) {

}

func (d dummyCluster) GetK8sConfig() ([]byte, error) {
	return d.K8sConfig, nil
}

func (d dummyCluster) GetName() string {
	return d.Name
}

func (d dummyCluster) GetOrganizationId() uint {
	return d.OrgID
}

func (d dummyCluster) GetUID() string {
	return d.UID
}

func (d dummyCluster) GetID() uint {
	return d.ID
}

func (d dummyCluster) IsReady() (bool, error) {
	return d.Ready, nil
}

func (d dummyCluster) NodePoolExists(nodePoolName string) bool {
	return d.NodePools[nodePoolName]
}

func (d dummyCluster) RbacEnabled() bool {
	return d.Rbac
}

type dummyOrgDomainService struct {
	Domain string
	OrgID  uint
}

func (dummyOrgDomainService) EnsureOrgDomain(ctx context.Context, clusterID uint) error {
	return nil
}

func (d dummyOrgDomainService) GetDomain(ctx context.Context, clusterID uint) (string, uint, error) {
	return d.Domain, d.OrgID, nil
}

type dummyOrganizationalSecretStore struct {
	Secrets map[uint]map[string]*secret.SecretItemResponse
}

func (d dummyOrganizationalSecretStore) Get(orgID uint, secretID string) (*secret.SecretItemResponse, error) {
	if orgSecrets, ok := d.Secrets[orgID]; ok {
		if sir, ok := orgSecrets[secretID]; ok {
			return sir, nil
		}
	}
	return nil, secret.ErrSecretNotExists
}

func (d dummyOrganizationalSecretStore) GetByName(organizationID uint, name string) (*secret.SecretItemResponse, error) {
	return &secret.SecretItemResponse{
		Name: name,
	}, nil
}

func (d dummyOrganizationalSecretStore) Store(organizationID uint, request *secret.CreateSecretRequest) (string, error) {
	return "randomsecretid", nil
}

func (d dummyOrganizationalSecretStore) Delete(organizationID uint, secretID string) error {
	return nil
}

type dummyHelmService struct {
}

func (d dummyHelmService) ApplyDeployment(
	ctx context.Context,
	clusterID uint,
	namespace string,
	deploymentName string,
	releaseName string,
	values []byte,
	chartVersion string,
) error {
	return nil
}

func (d dummyHelmService) DeleteDeployment(ctx context.Context, clusterID uint, releaseName string) error {
	return nil
}

func (d dummyHelmService) GetDeployment(ctx context.Context, clusterID uint, releaseName string) (*helm.GetDeploymentResponse, error) {
	return &helm.GetDeploymentResponse{
		ReleaseName: releaseName,
	}, nil
}
