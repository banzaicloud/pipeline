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

package cluster_test

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"testing"
	"time"

	"emperror.dev/emperror"
	"github.com/banzaicloud/bank-vaults/pkg/sdk/vault"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/global/nplabels"
	"github.com/banzaicloud/pipeline/internal/providers/azure/azureadapter"
	"github.com/banzaicloud/pipeline/internal/providers/kubernetes/kubernetesadapter"
	"github.com/banzaicloud/pipeline/internal/secret/pkesecret"
	"github.com/banzaicloud/pipeline/internal/secret/restricted"
	"github.com/banzaicloud/pipeline/internal/secret/secretadapter"
	"github.com/banzaicloud/pipeline/internal/secret/types"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/aks"
	"github.com/banzaicloud/pipeline/pkg/cluster/gke"
	"github.com/banzaicloud/pipeline/pkg/cluster/kubernetes"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	kubernetes2 "github.com/banzaicloud/pipeline/pkg/kubernetes"
	"github.com/banzaicloud/pipeline/src/cluster"
	"github.com/banzaicloud/pipeline/src/model"
	"github.com/banzaicloud/pipeline/src/secret"
)

const (
	clusterRequestName         = "testName"
	clusterRequestLocation     = "eu-west-1"
	clusterRequestNodeInstance = "testInstance"
	clusterRequestNodeCount    = 1
	clusterRequestRG           = "testResourceGroup"
	clusterRequestKubernetes   = "1.9.6"
	clusterRequestAgentName    = "testAgent"
	clusterRequestNodeMaxCount = 2
	organizationId             = 1
	userId                     = 1
	clusterKubeMetaKey         = "metaKey"
	clusterKubeMetaValue       = "metaValue"
	secretName                 = "test-secret-name"
	pool1Name                  = "pool1"
)

// nolint: gochecknoglobals
var (
	clusterRequestSecretId   = fmt.Sprintf("%x", sha256.Sum256([]byte(secretName)))
	clusterRequestNodeLabels = map[string]string{
		"testname": "testvalue",
	}
)

func TestIntegration(t *testing.T) {
	if m := flag.Lookup("test.run").Value.String(); m == "" || !regexp.MustCompile(m).MatchString(t.Name()) {
		t.Skip("skipping as execution was not requested explicitly using go test -run")
	}

	vaultAddr := os.Getenv("VAULT_ADDR")
	if vaultAddr == "" {
		t.Skip("skipping as VAULT_ADDR is not explicitly defined")
	}

	vaultClient, err := vault.NewClient("pipeline")
	emperror.Panic(err)
	global.SetVault(vaultClient)

	secretStore := secretadapter.NewVaultStore(vaultClient, "secret")
	pkeSecreter := pkesecret.NewPkeSecreter(vaultClient, common.NoopLogger{})
	secretTypes := types.NewDefaultTypeList(types.DefaultTypeListConfig{
		TLSDefaultValidity: 365 * 24 * time.Hour,
		PkeSecreter:        pkeSecreter,
	})
	secret.InitSecretStore(secretStore, secretTypes)
	restricted.InitSecretStore(secret.Store)

	t.Run("testCreateCommonClusterFromRequest", testCreateCommonClusterFromRequest)
	t.Run("testGKEKubernetesVersion", testGKEKubernetesVersion)
}

func testCreateCommonClusterFromRequest(t *testing.T) {
	labelValidator := kubernetes2.LabelValidator{
		ForbiddenDomains: []string{},
	}

	nplabels.SetNodePoolLabelValidator(labelValidator)

	cases := []struct {
		name          string
		createRequest *pkgCluster.CreateClusterRequest
		expectedModel *model.ClusterModel
		expectedError error
	}{
		{name: "aks create", createRequest: aksCreateFull, expectedModel: aksModelFull, expectedError: nil},
		{name: "kube create", createRequest: kubeCreateFull, expectedModel: kubeModelFull, expectedError: nil},

		{name: "not supported cloud", createRequest: notSupportedCloud, expectedModel: nil, expectedError: pkgErrors.ErrorNotSupportedCloudType},

		{name: "aks empty location", createRequest: aksEmptyLocationCreate, expectedModel: nil, expectedError: pkgErrors.ErrorLocationEmpty},
		{name: "kube empty location and nodeInstanceType", createRequest: kubeEmptyLocation, expectedModel: kubeEmptyLocAndNIT, expectedError: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			commonCluster, err := cluster.CreateCommonClusterFromRequest(tc.createRequest, organizationId, userId)

			if tc.expectedError != nil {
				if err != nil {
					if !reflect.DeepEqual(tc.expectedError, err) {
						t.Errorf("Expected model: %v, got: %v", tc.expectedError, err)
					}
				} else {
					t.Errorf("Expected error: %s, but not got error!", tc.expectedError.Error())
					t.FailNow()
				}
			} else {
				if err != nil {
					t.Errorf("Error during CreateCommonClusterFromRequest: %s", err.Error())
					t.FailNow()
				}

				modelAccessor, ok := commonCluster.(interface{ GetModel() *model.ClusterModel })
				if !ok {
					t.Fatal("model cannot be accessed")
				}

				if !reflect.DeepEqual(modelAccessor.GetModel(), tc.expectedModel) {
					t.Errorf("Expected model: %v, got: %v", tc.expectedModel, modelAccessor.GetModel())
				}
			}
		})
	}
}

func testGKEKubernetesVersion(t *testing.T) {
	testCases := []struct {
		name    string
		version string
		error
	}{
		{name: "version 1.5", version: "1.5", error: pkgErrors.ErrorWrongKubernetesVersion},
		{name: "version 1.6", version: "1.6", error: pkgErrors.ErrorWrongKubernetesVersion},
		{name: "version 1.7.7", version: "1.7.7", error: pkgErrors.ErrorWrongKubernetesVersion},
		{name: "version 1sd.8", version: "1sd", error: pkgErrors.ErrorWrongKubernetesVersion},
		{name: "version 1.8", version: "1.8", error: nil},
		{name: "version 1.82", version: "1.82", error: nil},
		{name: "version 1.9", version: "1.9", error: nil},
		{name: "version 1.15", version: "1.15", error: nil},
		{name: "version 2.0", version: "2.0", error: nil},
		{name: "version 2.3242.324", version: "2.3242.324", error: nil},
		{name: "version 11.5", version: "11.5", error: nil},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := gke.CreateClusterGKE{
				NodeVersion: tc.version,
				NodePools: map[string]*gke.NodePool{
					pool1Name: {
						Count:            clusterRequestNodeCount,
						NodeInstanceType: clusterRequestNodeInstance,
					},
				},
				Master: &gke.Master{
					Version: tc.version,
				},
			}

			err := g.Validate()

			if !reflect.DeepEqual(tc.error, err) {
				t.Errorf("Expected error: %#v, got: %#v", tc.error, err)
			}
		})
	}
}

// nolint: gochecknoglobals
var (
	aksCreateFull = &pkgCluster.CreateClusterRequest{
		Name:     clusterRequestName,
		Location: clusterRequestLocation,
		Cloud:    pkgCluster.Azure,
		SecretId: clusterRequestSecretId,
		Properties: &pkgCluster.CreateClusterProperties{
			CreateClusterAKS: &aks.CreateClusterAKS{
				ResourceGroup:     clusterRequestRG,
				KubernetesVersion: clusterRequestKubernetes,
				NodePools: map[string]*aks.NodePoolCreate{
					clusterRequestAgentName: {
						Autoscaling:      true,
						MinCount:         clusterRequestNodeCount,
						MaxCount:         clusterRequestNodeMaxCount,
						Count:            clusterRequestNodeCount,
						NodeInstanceType: clusterRequestNodeInstance,
						Labels:           clusterRequestNodeLabels,
					},
				},
			},
		},
	}

	aksEmptyLocationCreate = &pkgCluster.CreateClusterRequest{
		Name:     clusterRequestName,
		Location: "",
		Cloud:    pkgCluster.Azure,
		SecretId: clusterRequestSecretId,
		Properties: &pkgCluster.CreateClusterProperties{
			CreateClusterAKS: &aks.CreateClusterAKS{
				ResourceGroup:     clusterRequestRG,
				KubernetesVersion: clusterRequestKubernetes,
				NodePools: map[string]*aks.NodePoolCreate{
					clusterRequestAgentName: {
						Count:            clusterRequestNodeCount,
						NodeInstanceType: clusterRequestNodeInstance,
					},
				},
			},
		},
	}

	kubeCreateFull = &pkgCluster.CreateClusterRequest{
		Name:     clusterRequestName,
		Location: clusterRequestLocation,
		Cloud:    pkgCluster.Kubernetes,
		SecretId: clusterRequestSecretId,
		Properties: &pkgCluster.CreateClusterProperties{
			CreateClusterKubernetes: &kubernetes.CreateClusterKubernetes{
				Metadata: map[string]string{
					clusterKubeMetaKey: clusterKubeMetaValue,
				},
			},
		},
	}

	kubeEmptyLocation = &pkgCluster.CreateClusterRequest{
		Name:     clusterRequestName,
		Location: "",
		Cloud:    pkgCluster.Kubernetes,
		SecretId: clusterRequestSecretId,
		Properties: &pkgCluster.CreateClusterProperties{
			CreateClusterKubernetes: &kubernetes.CreateClusterKubernetes{
				Metadata: map[string]string{
					clusterKubeMetaKey: clusterKubeMetaValue,
				},
			},
		},
	}

	notSupportedCloud = &pkgCluster.CreateClusterRequest{
		Name:       clusterRequestName,
		Location:   clusterRequestLocation,
		Cloud:      "nonExistsCloud",
		SecretId:   clusterRequestSecretId,
		Properties: &pkgCluster.CreateClusterProperties{},
	}
)

// nolint: gochecknoglobals
var (
	aksModelFull = &model.ClusterModel{
		CreatedBy:      userId,
		Name:           clusterRequestName,
		Location:       clusterRequestLocation,
		SecretId:       clusterRequestSecretId,
		Cloud:          pkgCluster.Azure,
		Distribution:   pkgCluster.AKS,
		OrganizationId: organizationId,
		AKS: azureadapter.AKSClusterModel{
			ResourceGroup:     clusterRequestRG,
			KubernetesVersion: clusterRequestKubernetes,
			NodePools: []*azureadapter.AKSNodePoolModel{
				{
					CreatedBy:        userId,
					Autoscaling:      true,
					NodeMinCount:     clusterRequestNodeCount,
					NodeMaxCount:     clusterRequestNodeMaxCount,
					Count:            clusterRequestNodeCount,
					NodeInstanceType: clusterRequestNodeInstance,
					Name:             clusterRequestAgentName,
					Labels:           clusterRequestNodeLabels,
				},
			},
		},
	}

	kubeModelFull = &model.ClusterModel{
		CreatedBy:      userId,
		Name:           clusterRequestName,
		Location:       clusterRequestLocation,
		SecretId:       clusterRequestSecretId,
		Cloud:          pkgCluster.Kubernetes,
		Distribution:   pkgCluster.Unknown,
		OrganizationId: organizationId,
		Kubernetes: kubernetesadapter.KubernetesClusterModel{
			Metadata: map[string]string{
				clusterKubeMetaKey: clusterKubeMetaValue,
			},
			MetadataRaw: nil,
		},
	}

	kubeEmptyLocAndNIT = &model.ClusterModel{
		CreatedBy:      userId,
		Name:           clusterRequestName,
		Location:       "",
		SecretId:       clusterRequestSecretId,
		Cloud:          pkgCluster.Kubernetes,
		Distribution:   pkgCluster.Unknown,
		OrganizationId: organizationId,
		Kubernetes: kubernetesadapter.KubernetesClusterModel{
			Metadata: map[string]string{
				clusterKubeMetaKey: clusterKubeMetaValue,
			},
			MetadataRaw: nil,
		},
	}
)
