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
	"fmt"
	"reflect"
	"testing"

	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/aks"
	"github.com/banzaicloud/pipeline/pkg/cluster/dummy"
	"github.com/banzaicloud/pipeline/pkg/cluster/eks"
	"github.com/banzaicloud/pipeline/pkg/cluster/gke"
	"github.com/banzaicloud/pipeline/pkg/cluster/kubernetes"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/secret"
)

const (
	clusterRequestName          = "testName"
	clusterRequestLocation      = "testLocation"
	clusterRequestNodeInstance  = "testInstance"
	clusterRequestNodeCount     = 1
	clusterRequestRG            = "testResourceGroup"
	clusterRequestKubernetes    = "1.11.5"
	clusterRequestKubernetesEKS = "1.10"
	clusterRequestAgentName     = "testAgent"
	clusterRequestSpotPrice     = "1.2"
	clusterRequestNodeMinCount  = 1
	clusterRequestNodeMaxCount  = 2
	clusterRequestNodeImage     = "testImage"
	organizationId              = 1
	userId                      = 1
	clusterKubeMetaKey          = "metaKey"
	clusterKubeMetaValue        = "metaValue"
	secretName                  = "test-secret-name"
	pool1Name                   = "pool1"
)

var (
	clusterRequestSecretId   = fmt.Sprintf("%x", sha256.Sum256([]byte(secretName)))
	clusterRequestNodeLabels = map[string]string{
		"testname": "testvalue",
	}

	amazonSecretRequest = secret.CreateSecretRequest{
		Name: secretName,
		Type: pkgCluster.Amazon,
		Values: map[string]string{
			clusterKubeMetaKey: clusterKubeMetaValue,
		},
	}

	aksSecretRequest = secret.CreateSecretRequest{
		Name: secretName,
		Type: pkgCluster.Azure,
		Values: map[string]string{
			clusterKubeMetaKey: clusterKubeMetaValue,
		},
	}
)

var (
	errAzureAmazon = secret.MismatchError{
		SecretType: pkgCluster.Azure,
		ValidType:  pkgCluster.Amazon,
	}
)

func TestCreateCommonClusterFromRequest(t *testing.T) {

	cases := []struct {
		name          string
		createRequest *pkgCluster.CreateClusterRequest
		expectedModel *model.ClusterModel
		expectedError error
	}{
		{name: "aks create", createRequest: aksCreateFull, expectedModel: aksModelFull, expectedError: nil},
		{name: "dummy create", createRequest: dummyCreateFull, expectedModel: dummyModelFull, expectedError: nil},
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

func TestGKEKubernetesVersion(t *testing.T) {

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

func TestGetSecretWithValidation(t *testing.T) {

	cases := []struct {
		name                 string
		secretRequest        secret.CreateSecretRequest
		createClusterRequest *pkgCluster.CreateClusterRequest
		err                  error
	}{
		{"amazon", amazonSecretRequest, eksCreateFull, nil},
		{"aks", aksSecretRequest, aksCreateFull, nil},
		{"aks wrong cloud field", aksSecretRequest, eksCreateFull, errAzureAmazon},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			if secretID, err := secret.Store.Store(organizationId, &tc.secretRequest); err != nil {
				t.Errorf("Error during saving secret: %s", err.Error())
				t.FailNow()
			} else {
				defer secret.Store.Delete(organizationId, secretID)
			}

			commonCluster, err := cluster.CreateCommonClusterFromRequest(tc.createClusterRequest, organizationId, userId)
			if err != nil {
				t.Errorf("Error during create model from request: %s", err.Error())
				t.FailNow()
			}

			_, err = commonCluster.GetSecretWithValidation()
			if tc.err != nil {
				if err == nil {
					t.Errorf("Expected error: %s, but got non", tc.err.Error())
					t.FailNow()
				} else if !reflect.DeepEqual(tc.err, err) {
					t.Errorf("Expected error: %s, but got: %s", tc.err.Error(), err.Error())
					t.FailNow()
				}
			} else if err != nil {
				t.Errorf("Error during secret validation: %v", err)
				t.FailNow()
			}
		})
	}

}

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

	eksCreateFull = &pkgCluster.CreateClusterRequest{ // nolint deadcode
		Name:     clusterRequestName,
		Location: clusterRequestLocation,
		Cloud:    pkgCluster.Amazon,
		SecretId: clusterRequestSecretId,
		Properties: &pkgCluster.CreateClusterProperties{
			CreateClusterEKS: &eks.CreateClusterEKS{
				Version: clusterRequestKubernetesEKS,
				NodePools: map[string]*eks.NodePool{
					pool1Name: {
						InstanceType: clusterRequestNodeInstance,
						SpotPrice:    clusterRequestSpotPrice,
						Autoscaling:  true,
						MinCount:     clusterRequestNodeMinCount,
						MaxCount:     clusterRequestNodeMaxCount,
						Count:        clusterRequestNodeCount,
						Image:        clusterRequestNodeImage,
					},
				},
			},
		},
	}

	dummyCreateFull = &pkgCluster.CreateClusterRequest{
		Name:     clusterRequestName,
		Location: clusterRequestLocation,
		Cloud:    pkgCluster.Dummy,
		SecretId: clusterRequestSecretId,
		Properties: &pkgCluster.CreateClusterProperties{
			CreateClusterDummy: &dummy.CreateClusterDummy{
				Node: &dummy.Node{
					KubernetesVersion: clusterRequestKubernetes,
					Count:             clusterRequestNodeCount,
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

var (
	aksModelFull = &model.ClusterModel{
		CreatedBy:      userId,
		Name:           clusterRequestName,
		Location:       clusterRequestLocation,
		SecretId:       clusterRequestSecretId,
		Cloud:          pkgCluster.Azure,
		Distribution:   pkgCluster.AKS,
		OrganizationId: organizationId,
		AKS: model.AKSClusterModel{
			ResourceGroup:     clusterRequestRG,
			KubernetesVersion: clusterRequestKubernetes,
			NodePools: []*model.AKSNodePoolModel{
				{
					CreatedBy:        userId,
					Autoscaling:      true,
					NodeMinCount:     clusterRequestNodeCount,
					NodeMaxCount:     clusterRequestNodeMaxCount,
					Count:            clusterRequestNodeCount,
					NodeInstanceType: clusterRequestNodeInstance,
					Name:             clusterRequestAgentName,
					Labels: []*model.AKSNodePoolLabelModel{
						{
							Name:  "testname",
							Value: "testvalue",
						},
					},
				},
			},
		},
	}

	dummyModelFull = &model.ClusterModel{
		CreatedBy:      userId,
		Name:           clusterRequestName,
		Location:       clusterRequestLocation,
		Cloud:          pkgCluster.Dummy,
		Distribution:   pkgCluster.Dummy,
		OrganizationId: organizationId,
		SecretId:       clusterRequestSecretId,
		Dummy: model.DummyClusterModel{
			KubernetesVersion: clusterRequestKubernetes,
			NodeCount:         clusterRequestNodeCount,
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
		Kubernetes: model.KubernetesClusterModel{
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
		Kubernetes: model.KubernetesClusterModel{
			Metadata: map[string]string{
				clusterKubeMetaKey: clusterKubeMetaValue,
			},
			MetadataRaw: nil,
		},
	}
)
