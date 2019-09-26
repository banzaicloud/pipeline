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

package monitoring

import (
	"context"

	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/secret"
)

type obj = map[string]interface{}

const (
	grafanaSecretID    = "grafanaSecretID"
	prometheusSecretID = "prometheusSecretID"
	grafanaPath        = "/grafana"
	prometheusPath     = "/prometheus"
	grafanaURL         = "http://monitoring.io/grafana"
	prometheusURL      = "http://monitoring.io/prometheus"
)

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

func (d dummyOrganizationalSecretStore) Store(organizationID uint, request *secret.CreateSecretRequest) (string, error) {
	return prometheusSecretID, nil
}

func (d dummyOrganizationalSecretStore) GetByName(organizationID uint, name string) (*secret.SecretItemResponse, error) {
	if orgSecrets, ok := d.Secrets[organizationID]; ok {
		for n, sir := range orgSecrets {
			if n == name {
				return sir, nil
			}
		}
	}
	return nil, secret.ErrSecretNotExists
}

func (d dummyOrganizationalSecretStore) Delete(organizationID uint, secretID string) error {
	return nil
}

type dummyEndpointService struct{}

func (dummyEndpointService) List(kubeConfig []byte, releaseName string) ([]*helm.EndpointItem, error) {
	return []*helm.EndpointItem{
		{
			Name: "ingress-traefik",
			EndPointURLs: []*helm.EndPointURLs{
				{
					Path:        grafanaPath,
					URL:         grafanaURL,
					ReleaseName: releaseName,
				},
				{
					Path:        prometheusPath,
					URL:         prometheusURL,
					ReleaseName: releaseName,
				},
			},
		},
	}, nil
}

type dummyHelmService struct{}

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
