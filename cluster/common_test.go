package cluster_test

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"testing"

	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"

	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/pkg/cluster/amazon"
	"github.com/banzaicloud/pipeline/pkg/cluster/azure"
	"github.com/banzaicloud/pipeline/pkg/cluster/dummy"
	"github.com/banzaicloud/pipeline/pkg/cluster/google"
	"github.com/banzaicloud/pipeline/pkg/cluster/kubernetes"
	"github.com/banzaicloud/pipeline/secret"
)

const (
	clusterRequestName           = "testName"
	clusterRequestLocation       = "testLocation"
	clusterRequestNodeInstance   = "testInstance"
	clusterRequestNodeCount      = 1
	clusterRequestVersion        = "1.9.4-gke.1"
	clusterRequestVersion2       = "1.8.7-gke.2"
	clusterRequestWrongVersion   = "1.7.7-gke.1"
	clusterRequestRG             = "testResourceGroup"
	clusterRequestKubernetes     = "1.9.2"
	clusterRequestAgentName      = "testAgent"
	clusterRequestSpotPrice      = "1.2"
	clusterRequestNodeMaxCount   = 2
	clusterRequestNodeImage      = "testImage"
	clusterRequestMasterImage    = "testImage"
	clusterRequestMasterInstance = "testInstance"
	clusterServiceAccount        = "testServiceAccount"
	organizationId               = 1
	clusterKubeMetaKey           = "metaKey"
	clusterKubeMetaValue         = "metaValue"
	secretName                   = "test-secret-name"
	pool1Name                    = "pool1"
)

var (
	clusterRequestSecretId = fmt.Sprintf("%x", sha256.Sum256([]byte(secretName)))

	awsSecretRequest = secret.CreateSecretRequest{
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

	gkeSecretRequest = secret.CreateSecretRequest{
		Name: secretName,
		Type: pkgCluster.Google,
		Values: map[string]string{
			clusterKubeMetaKey: clusterKubeMetaValue,
		},
	}
)

var (
	errAmazonGoogle = secret.MissmatchError{
		SecretType: pkgCluster.Amazon,
		ValidType:  pkgCluster.Google,
	}

	errAzureAmazon = secret.MissmatchError{
		SecretType: pkgCluster.Azure,
		ValidType:  pkgCluster.Amazon,
	}

	errGoogleAmazon = secret.MissmatchError{
		SecretType: pkgCluster.Google,
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
		{name: "gke create", createRequest: gkeCreateFull, expectedModel: gkeModelFull, expectedError: nil},
		{name: "aks create", createRequest: aksCreateFull, expectedModel: aksModelFull, expectedError: nil},
		{name: "aws create", createRequest: awsCreateFull, expectedModel: awsModelFull, expectedError: nil},
		{name: "dummy create", createRequest: dummyCreateFull, expectedModel: dummyModelFull, expectedError: nil},
		{name: "kube create", createRequest: kubeCreateFull, expectedModel: kubeModelFull, expectedError: nil},

		{name: "gke wrong k8s version", createRequest: gkeWrongK8sVersion, expectedModel: nil, expectedError: pkgErrors.ErrorWrongKubernetesVersion},
		{name: "gke different k8s version", createRequest: gkeDifferentK8sVersion, expectedModel: gkeModelDifferentVersion, expectedError: constants.ErrorDifferentKubernetesVersion},

		{name: "not supported cloud", createRequest: notSupportedCloud, expectedModel: nil, expectedError: pkgErrors.ErrorNotSupportedCloudType},

		{name: "aws empty location", createRequest: awsEmptyLocationCreate, expectedModel: nil, expectedError: pkgErrors.ErrorLocationEmpty},
		{name: "aks empty location", createRequest: aksEmptyLocationCreate, expectedModel: nil, expectedError: pkgErrors.ErrorLocationEmpty},
		{name: "gke empty location", createRequest: gkeEmptyLocationCreate, expectedModel: nil, expectedError: pkgErrors.ErrorLocationEmpty},
		{name: "kube empty location and nodeInstanceType", createRequest: kubeEmptyLocation, expectedModel: kubeEmptyLocAndNIT, expectedError: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			commonCluster, err := cluster.CreateCommonClusterFromRequest(tc.createRequest, organizationId)

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

				if !reflect.DeepEqual(commonCluster.GetModel(), tc.expectedModel) {
					t.Errorf("Expected model: %v, got: %v", tc.expectedModel, commonCluster.GetModel())
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
			g := google.CreateClusterGoogle{
				NodeVersion: tc.version,
				NodePools: map[string]*google.NodePool{
					pool1Name: {
						Count:            clusterRequestNodeCount,
						NodeInstanceType: clusterRequestNodeInstance,
						ServiceAccount:   clusterServiceAccount,
					},
				},
				Master: &google.Master{
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
		{"aws", awsSecretRequest, awsCreateFull, nil},
		{"aks", aksSecretRequest, aksCreateFull, nil},
		{"gke", gkeSecretRequest, gkeCreateFull, nil},
		{"aws wrong cloud field", awsSecretRequest, gkeCreateFull, errAmazonGoogle},
		{"aks wrong cloud field", aksSecretRequest, awsCreateFull, errAzureAmazon},
		{"gke wrong cloud field", gkeSecretRequest, awsCreateFull, errGoogleAmazon},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			if secretID, err := secret.Store.Store(organizationId, &tc.secretRequest); err != nil {
				t.Errorf("Error during saving secret: %s", err.Error())
				t.FailNow()
			} else {
				defer secret.Store.Delete(organizationId, secretID)
			}

			commonCluster, err := cluster.CreateCommonClusterFromRequest(tc.createClusterRequest, organizationId)
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
	gkeCreateFull = &pkgCluster.CreateClusterRequest{
		Name:     clusterRequestName,
		Location: clusterRequestLocation,
		Cloud:    pkgCluster.Google,
		SecretId: clusterRequestSecretId,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon  `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure    `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle  `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy    `json:"dummy,omitempty"`
			CreateKubernetes    *kubernetes.CreateKubernetes `json:"kubernetes,omitempty"`
		}{
			CreateClusterGoogle: &google.CreateClusterGoogle{
				NodeVersion: clusterRequestVersion,
				NodePools: map[string]*google.NodePool{
					pool1Name: {
						Autoscaling:      true,
						MinCount:         clusterRequestNodeCount,
						MaxCount:         clusterRequestNodeMaxCount,
						Count:            clusterRequestNodeCount,
						NodeInstanceType: clusterRequestNodeInstance,
						ServiceAccount:   clusterServiceAccount,
					},
				},
				Master: &google.Master{
					Version: clusterRequestVersion,
				},
			},
		},
	}

	gkeEmptyLocationCreate = &pkgCluster.CreateClusterRequest{
		Name:     clusterRequestName,
		Location: "",
		Cloud:    constants.Google,
		SecretId: clusterRequestSecretId,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon  `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure    `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle  `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy    `json:"dummy,omitempty"`
			CreateKubernetes    *kubernetes.CreateKubernetes `json:"kubernetes,omitempty"`
		}{
			CreateClusterGoogle: &google.CreateClusterGoogle{
				NodeVersion: clusterRequestVersion,
				NodePools: map[string]*google.NodePool{
					pool1Name: {
						Count:            clusterRequestNodeCount,
						NodeInstanceType: clusterRequestNodeInstance,
						ServiceAccount:   clusterServiceAccount,
					},
				},
				Master: &google.Master{
					Version: clusterRequestVersion,
				},
			},
		},
	}

	aksCreateFull = &pkgCluster.CreateClusterRequest{
		Name:     clusterRequestName,
		Location: clusterRequestLocation,
		Cloud:    pkgCluster.Azure,
		SecretId: clusterRequestSecretId,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon  `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure    `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle  `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy    `json:"dummy,omitempty"`
			CreateKubernetes    *kubernetes.CreateKubernetes `json:"kubernetes,omitempty"`
		}{
			CreateClusterAzure: &azure.CreateClusterAzure{
				ResourceGroup:     clusterRequestRG,
				KubernetesVersion: clusterRequestKubernetes,
				NodePools: map[string]*azure.NodePoolCreate{
					clusterRequestAgentName: {
						Autoscaling:      true,
						MinCount:         clusterRequestNodeCount,
						MaxCount:         clusterRequestNodeMaxCount,
						Count:            clusterRequestNodeCount,
						NodeInstanceType: clusterRequestNodeInstance,
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
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon  `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure    `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle  `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy    `json:"dummy,omitempty"`
			CreateKubernetes    *kubernetes.CreateKubernetes `json:"kubernetes,omitempty"`
		}{
			CreateClusterAzure: &azure.CreateClusterAzure{
				ResourceGroup:     clusterRequestRG,
				KubernetesVersion: clusterRequestKubernetes,
				NodePools: map[string]*azure.NodePoolCreate{
					clusterRequestAgentName: {
						Count:            clusterRequestNodeCount,
						NodeInstanceType: clusterRequestNodeInstance,
					},
				},
			},
		},
	}

	awsCreateFull = &pkgCluster.CreateClusterRequest{
		Name:     clusterRequestName,
		Location: clusterRequestLocation,
		Cloud:    pkgCluster.Amazon,
		SecretId: clusterRequestSecretId,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon  `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure    `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle  `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy    `json:"dummy,omitempty"`
			CreateKubernetes    *kubernetes.CreateKubernetes `json:"kubernetes,omitempty"`
		}{
			CreateClusterAmazon: &amazon.CreateClusterAmazon{
				NodePools: map[string]*amazon.NodePool{
					pool1Name: {
						InstanceType: clusterRequestNodeInstance,
						SpotPrice:    clusterRequestSpotPrice,
						Autoscaling:  true,
						MinCount:     clusterRequestNodeCount,
						MaxCount:     clusterRequestNodeMaxCount,
						Image:        clusterRequestNodeImage,
					},
				},
				Master: &amazon.CreateAmazonMaster{
					InstanceType: clusterRequestMasterInstance,
					Image:        clusterRequestMasterImage,
				},
			},
		},
	}

	dummyCreateFull = &pkgCluster.CreateClusterRequest{
		Name:     clusterRequestName,
		Location: clusterRequestLocation,
		Cloud:    pkgCluster.Dummy,
		SecretId: clusterRequestSecretId,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon  `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure    `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle  `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy    `json:"dummy,omitempty"`
			CreateKubernetes    *kubernetes.CreateKubernetes `json:"kubernetes,omitempty"`
		}{
			CreateClusterDummy: &dummy.CreateClusterDummy{
				Node: &dummy.Node{
					KubernetesVersion: clusterRequestKubernetes,
					Count:             clusterRequestNodeCount,
				},
			},
		},
	}

	awsEmptyLocationCreate = &pkgCluster.CreateClusterRequest{
		Name:     clusterRequestName,
		Location: "",
		Cloud:    pkgCluster.Amazon,
		SecretId: clusterRequestSecretId,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon  `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure    `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle  `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy    `json:"dummy,omitempty"`
			CreateKubernetes    *kubernetes.CreateKubernetes `json:"kubernetes,omitempty"`
		}{
			CreateClusterAmazon: &amazon.CreateClusterAmazon{
				NodePools: map[string]*amazon.NodePool{
					pool1Name: {
						InstanceType: clusterRequestNodeInstance,
						SpotPrice:    clusterRequestSpotPrice,
						MinCount:     clusterRequestNodeCount,
						MaxCount:     clusterRequestNodeMaxCount,
						Image:        clusterRequestNodeImage,
					},
				},
				Master: &amazon.CreateAmazonMaster{
					InstanceType: clusterRequestMasterInstance,
					Image:        clusterRequestMasterImage,
				},
			},
		},
	}

	kubeCreateFull = &pkgCluster.CreateClusterRequest{
		Name:     clusterRequestName,
		Location: clusterRequestLocation,
		Cloud:    pkgCluster.Kubernetes,
		SecretId: clusterRequestSecretId,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon  `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure    `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle  `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy    `json:"dummy,omitempty"`
			CreateKubernetes    *kubernetes.CreateKubernetes `json:"kubernetes,omitempty"`
		}{
			CreateKubernetes: &kubernetes.CreateKubernetes{
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
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon  `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure    `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle  `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy    `json:"dummy,omitempty"`
			CreateKubernetes    *kubernetes.CreateKubernetes `json:"kubernetes,omitempty"`
		}{
			CreateKubernetes: &kubernetes.CreateKubernetes{
				Metadata: map[string]string{
					clusterKubeMetaKey: clusterKubeMetaValue,
				},
			},
		},
	}

	notSupportedCloud = &pkgCluster.CreateClusterRequest{
		Name:     clusterRequestName,
		Location: clusterRequestLocation,
		Cloud:    "nonExistsCloud",
		SecretId: clusterRequestSecretId,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon  `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure    `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle  `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy    `json:"dummy,omitempty"`
			CreateKubernetes    *kubernetes.CreateKubernetes `json:"kubernetes,omitempty"`
		}{},
	}

	gkeWrongK8sVersion = &pkgCluster.CreateClusterRequest{
		Name:     clusterRequestName,
		Location: clusterRequestLocation,
		Cloud:    pkgCluster.Google,
		SecretId: clusterRequestSecretId,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon  `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure    `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle  `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy    `json:"dummy,omitempty"`
			CreateKubernetes    *kubernetes.CreateKubernetes `json:"kubernetes,omitempty"`
		}{
			CreateClusterGoogle: &google.CreateClusterGoogle{
				NodeVersion: clusterRequestVersion,
				NodePools: map[string]*google.NodePool{
					pool1Name: {
						Count:            clusterRequestNodeCount,
						NodeInstanceType: clusterRequestNodeInstance,
						ServiceAccount:   clusterServiceAccount,
					},
				},
				Master: &google.Master{
					Version: clusterRequestWrongVersion,
				},
			},
		},
	}

	gkeDifferentK8sVersion = &pkgCluster.CreateClusterRequest{
		Name:     clusterRequestName,
		Location: clusterRequestLocation,
		Cloud:    pkgCluster.Google,
		SecretId: clusterRequestSecretId,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon  `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure    `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle  `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy    `json:"dummy,omitempty"`
			CreateKubernetes    *kubernetes.CreateKubernetes `json:"kubernetes,omitempty"`
		}{
			CreateClusterGoogle: &google.CreateClusterGoogle{
				NodeVersion: clusterRequestVersion,
				NodePools: map[string]*google.NodePool{
					pool1Name: {
						Count:            clusterRequestNodeCount,
						NodeInstanceType: clusterRequestNodeInstance,
						ServiceAccount:   clusterServiceAccount,
					},
				},
				Master: &google.Master{
					Version: clusterRequestVersion2,
				},
			},
		},
	}
)

var (
	gkeModelFull = &model.ClusterModel{
		Name:           clusterRequestName,
		Location:       clusterRequestLocation,
		SecretId:       clusterRequestSecretId,
		Cloud:          pkgCluster.Google,
		OrganizationId: organizationId,
		Amazon:         model.AmazonClusterModel{},
		Azure:          model.AzureClusterModel{},
		Google: model.GoogleClusterModel{
			MasterVersion: clusterRequestVersion,
			NodeVersion:   clusterRequestVersion,
			NodePools: []*model.GoogleNodePoolModel{
				{
					Name:             pool1Name,
					Autoscaling:      true,
					NodeMinCount:     clusterRequestNodeCount,
					NodeMaxCount:     clusterRequestNodeMaxCount,
					NodeCount:        clusterRequestNodeCount,
					NodeInstanceType: clusterRequestNodeInstance,
					ServiceAccount:   clusterServiceAccount,
				},
			},
		},
	}

	aksModelFull = &model.ClusterModel{
		Name:           clusterRequestName,
		Location:       clusterRequestLocation,
		SecretId:       clusterRequestSecretId,
		Cloud:          pkgCluster.Azure,
		OrganizationId: organizationId,
		Amazon:         model.AmazonClusterModel{},
		Azure: model.AzureClusterModel{
			ResourceGroup:     clusterRequestRG,
			KubernetesVersion: clusterRequestKubernetes,
			NodePools: []*model.AzureNodePoolModel{
				{
					Autoscaling:      true,
					NodeMinCount:     clusterRequestNodeCount,
					NodeMaxCount:     clusterRequestNodeMaxCount,
					Count:            clusterRequestNodeCount,
					NodeInstanceType: clusterRequestNodeInstance,
					Name:             clusterRequestAgentName,
				},
			},
		},
		Google: model.GoogleClusterModel{},
	}

	awsModelFull = &model.ClusterModel{
		Name:           clusterRequestName,
		Location:       clusterRequestLocation,
		SecretId:       clusterRequestSecretId,
		Cloud:          pkgCluster.Amazon,
		OrganizationId: organizationId,
		Amazon: model.AmazonClusterModel{
			NodePools: []*model.AmazonNodePoolsModel{
				{
					Name:             pool1Name,
					NodeInstanceType: clusterRequestNodeInstance,
					NodeSpotPrice:    clusterRequestSpotPrice,
					Autoscaling:      true,
					Count:            clusterRequestNodeCount,
					NodeMinCount:     clusterRequestNodeCount,
					NodeMaxCount:     clusterRequestNodeMaxCount,
					NodeImage:        clusterRequestNodeImage,
				}},
			MasterInstanceType: clusterRequestMasterInstance,
			MasterImage:        clusterRequestMasterImage,
		},
		Azure:  model.AzureClusterModel{},
		Google: model.GoogleClusterModel{},
	}

	dummyModelFull = &model.ClusterModel{
		Name:           clusterRequestName,
		Location:       clusterRequestLocation,
		Cloud:          pkgCluster.Dummy,
		OrganizationId: organizationId,
		SecretId:       clusterRequestSecretId,
		Amazon:         model.AmazonClusterModel{},
		Azure:          model.AzureClusterModel{},
		Google:         model.GoogleClusterModel{},
		Dummy: model.DummyClusterModel{
			KubernetesVersion: clusterRequestKubernetes,
			NodeCount:         clusterRequestNodeCount,
		},
	}

	kubeModelFull = &model.ClusterModel{
		Name:           clusterRequestName,
		Location:       clusterRequestLocation,
		SecretId:       clusterRequestSecretId,
		Cloud:          pkgCluster.Kubernetes,
		OrganizationId: organizationId,
		Amazon:         model.AmazonClusterModel{},
		Azure:          model.AzureClusterModel{},
		Google:         model.GoogleClusterModel{},
		Kubernetes: model.KubernetesClusterModel{
			Metadata: map[string]string{
				clusterKubeMetaKey: clusterKubeMetaValue,
			},
			MetadataRaw: nil,
		},
	}

	kubeEmptyLocAndNIT = &model.ClusterModel{
		Name:           clusterRequestName,
		Location:       "",
		SecretId:       clusterRequestSecretId,
		Cloud:          pkgCluster.Kubernetes,
		OrganizationId: organizationId,
		Amazon:         model.AmazonClusterModel{},
		Azure:          model.AzureClusterModel{},
		Google:         model.GoogleClusterModel{},
		Kubernetes: model.KubernetesClusterModel{
			Metadata: map[string]string{
				clusterKubeMetaKey: clusterKubeMetaValue,
			},
			MetadataRaw: nil,
		},
	}

	gkeModelDifferentVersion = &model.ClusterModel{
		Name:           clusterRequestName,
		Location:       clusterRequestLocation,
		SecretId:       clusterRequestSecretId,
		Cloud:          pkgCluster.Google,
		OrganizationId: organizationId,
		Amazon:         model.AmazonClusterModel{},
		Azure:          model.AzureClusterModel{},
		Google: model.GoogleClusterModel{
			MasterVersion: clusterRequestVersion2,
			NodeVersion:   clusterRequestVersion,
			NodePools: []*model.GoogleNodePoolModel{
				{
					Name:             pool1Name,
					NodeCount:        clusterRequestNodeCount,
					NodeInstanceType: clusterRequestNodeInstance,
					ServiceAccount:   clusterServiceAccount,
				},
			},
		},
	}
)
