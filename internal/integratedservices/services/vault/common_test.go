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

package vault

import (
	"context"

	"emperror.dev/errors"
	"github.com/dgrijalva/jwt-go"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8srest "k8s.io/client-go/rest"

	"github.com/banzaicloud/pipeline/internal/integratedservices/integratedserviceadapter"
	"github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/src/secret"
)

type obj = map[string]interface{}

const tokenSecretID = "vaulttokensecretid"

type dummyClusterGetter struct {
	Clusters map[uint]dummyCluster
}

func (d dummyClusterGetter) GetClusterByIDOnly(ctx context.Context, clusterID uint) (integratedserviceadapter.Cluster, error) {
	return d.Clusters[clusterID], nil
}

func (d dummyClusterGetter) GetClusterStatus(ctx context.Context, clusterID uint) (string, error) {
	if c, ok := d.Clusters[clusterID]; ok {
		return c.Status, nil
	}
	return "", errors.New("cluster not found")
}

type dummyCluster struct {
	K8sConfig []byte
	Name      string
	OrgID     uint
	ID        uint
	UID       string
	NodePools map[string]bool
	Rbac      bool
	Status    string
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
	return tokenSecretID, nil
}

func (d dummyOrganizationalSecretStore) GetByName(organizationID uint, name string) (*secret.SecretItemResponse, error) {
	return &secret.SecretItemResponse{
		Name: name,
	}, nil
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

func (d dummyHelmService) DeleteDeployment(ctx context.Context, clusterID uint, releaseName, namespace string) error {
	return nil
}

func (d dummyHelmService) GetDeployment(ctx context.Context, clusterID uint, releaseName, namespace string) (*helm.GetDeploymentResponse, error) {
	return &helm.GetDeploymentResponse{
		ReleaseName: releaseName,
	}, nil
}

type dummyKubernetesService struct {
}

// GetKubeConfig gets a kube config for a specific cluster.
func (s *dummyKubernetesService) GetKubeConfig(ctx context.Context, clusterID uint) (*k8srest.Config, error) {
	return &k8srest.Config{
		Host:            "https://127.0.0.1:6443",
		TLSClientConfig: k8srest.TLSClientConfig{CAData: []byte("BLABLA")},
	}, nil
}

// DeleteObject deletes an Object from a specific cluster.
func (s *dummyKubernetesService) DeleteObject(ctx context.Context, clusterID uint, o runtime.Object) error {
	return nil
}

// EnsureObject makes sure that a given Object is on the cluster and returns it.
func (s *dummyKubernetesService) EnsureObject(ctx context.Context, clusterID uint, o runtime.Object) error {
	switch v := o.(type) {
	case *corev1.ServiceAccount:
		v.Secrets = []corev1.ObjectReference{{Name: "some-token-1234", Namespace: "default"}}
	case *corev1.Secret:
		token, err := jwt.New(jwt.SigningMethodHS256).SignedString([]byte("random-key"))
		if err != nil {
			return err
		}

		v.Data = map[string][]byte{corev1.ServiceAccountTokenKey: []byte(token)}
	}

	return nil
}
