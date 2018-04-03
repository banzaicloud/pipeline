package cluster_test

import (
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/components/amazon"
	"github.com/banzaicloud/banzai-types/components/azure"
	"github.com/banzaicloud/banzai-types/components/dummy"
	"github.com/banzaicloud/banzai-types/components/byoc"
	"github.com/banzaicloud/banzai-types/components/google"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/model"
	"reflect"
	"testing"
)

const (
	clusterRequestName           = "testName"
	clusterRequestLocation       = "testLocation"
	clusterRequestNodeInstance   = "testInstance"
	clusterRequestSecretId       = ""
	clusterRequestProject        = "testProject"
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
	clusterBYOCMetaKey           = "metaKey"
	clusterBYOCMetaValue         = "metaValue"
)

func TestCreateCommonClusterFromRequest(t *testing.T) {

	cases := []struct {
		name          string
		createRequest *components.CreateClusterRequest
		expectedModel *model.ClusterModel
		expectedError error
	}{
		{name: "gke create", createRequest: gkeCreateFull, expectedModel: gkeModelFull, expectedError: nil},
		{name: "aks create", createRequest: aksCreateFull, expectedModel: aksModelFull, expectedError: nil},
		{name: "aws create", createRequest: awsCreateFull, expectedModel: awsModelFull, expectedError: nil},
		{name: "dummy create", createRequest: dummyCreateFull, expectedModel: dummyModelFull, expectedError: nil},
		{name: "byoc create", createRequest: byocCreateFull, expectedModel: byocModelFull, expectedError: nil},

		{name: "gke wrong k8s version", createRequest: gkeWrongK8sVersion, expectedModel: nil, expectedError: constants.ErrorWrongKubernetesVersion},
		{name: "gke different k8s version", createRequest: gkeDifferentK8sVersion, expectedModel: gkeModelDifferentVersion, expectedError: constants.ErrorDifferentKubernetesVersion},

		{name: "not supported cloud", createRequest: notSupportedCloud, expectedModel: nil, expectedError: constants.ErrorNotSupportedCloudType},

		{name: "aws empty location", createRequest: awsEmptyLocationCreate, expectedModel: nil, expectedError: constants.ErrorLocationEmpty},
		{name: "aws empty nodeInstanceType", createRequest: awsEmptyNITCreate, expectedModel: nil, expectedError: constants.ErrorNodeInstanceTypeEmpty},
		{name: "aks empty location", createRequest: aksEmptyLocationCreate, expectedModel: nil, expectedError: constants.ErrorLocationEmpty},
		{name: "aks empty nodeInstanceType", createRequest: aksEmptyNITCreate, expectedModel: nil, expectedError: constants.ErrorNodeInstanceTypeEmpty},
		{name: "gke empty location", createRequest: gkeEmptyLocationCreate, expectedModel: nil, expectedError: constants.ErrorLocationEmpty},
		{name: "gke empty nodeInstanceType", createRequest: gkeEmptyNITCreate, expectedModel: nil, expectedError: constants.ErrorNodeInstanceTypeEmpty},
		{name: "byoc empty location and nodeInstanceType", createRequest: byocEmptyLocationAndNIT, expectedModel: byocEmptyLocAndNIT, expectedError: nil},
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
		{name: "version 1.5", version: "1.5", error: constants.ErrorWrongKubernetesVersion},
		{name: "version 1.6", version: "1.6", error: constants.ErrorWrongKubernetesVersion},
		{name: "version 1.7.7", version: "1.7.7", error: constants.ErrorWrongKubernetesVersion},
		{name: "version 1sd.8", version: "1sd", error: constants.ErrorWrongKubernetesVersion},
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
				Project: clusterRequestProject,
				Node: &google.GoogleNode{
					Count:          clusterRequestNodeCount,
					Version:        tc.version,
					ServiceAccount: clusterServiceAccount,
				},
				Master: &google.GoogleMaster{
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

var (
	gkeCreateFull = &components.CreateClusterRequest{
		Name:             clusterRequestName,
		Location:         clusterRequestLocation,
		Cloud:            constants.Google,
		NodeInstanceType: clusterRequestNodeInstance,
		SecretId:         clusterRequestSecretId,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy   `json:"dummy,omitempty"`
			CreateBYOC          *byoc.CreateBYOC            `json:"byoc,omitempty"`
		}{
			CreateClusterGoogle: &google.CreateClusterGoogle{
				Project: clusterRequestProject,
				Node: &google.GoogleNode{
					Count:          clusterRequestNodeCount,
					Version:        clusterRequestVersion,
					ServiceAccount: clusterServiceAccount,
				},
				Master: &google.GoogleMaster{
					Version: clusterRequestVersion,
				},
			},
		},
	}

	gkeEmptyLocationCreate = &components.CreateClusterRequest{
		Name:             clusterRequestName,
		Location:         "",
		Cloud:            constants.Google,
		NodeInstanceType: clusterRequestNodeInstance,
		SecretId:         clusterRequestSecretId,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy   `json:"dummy,omitempty"`
			CreateBYOC          *byoc.CreateBYOC            `json:"byoc,omitempty"`
		}{
			CreateClusterGoogle: &google.CreateClusterGoogle{
				Project: clusterRequestProject,
				Node: &google.GoogleNode{
					Count:          clusterRequestNodeCount,
					Version:        clusterRequestVersion,
					ServiceAccount: clusterServiceAccount,
				},
				Master: &google.GoogleMaster{
					Version: clusterRequestVersion,
				},
			},
		},
	}

	gkeEmptyNITCreate = &components.CreateClusterRequest{
		Name:             clusterRequestName,
		Location:         clusterRequestLocation,
		Cloud:            constants.Google,
		NodeInstanceType: "",
		SecretId:         clusterRequestSecretId,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy   `json:"dummy,omitempty"`
			CreateBYOC          *byoc.CreateBYOC            `json:"byoc,omitempty"`
		}{
			CreateClusterGoogle: &google.CreateClusterGoogle{
				Project: clusterRequestProject,
				Node: &google.GoogleNode{
					Count:          clusterRequestNodeCount,
					Version:        clusterRequestVersion,
					ServiceAccount: clusterServiceAccount,
				},
				Master: &google.GoogleMaster{
					Version: clusterRequestVersion,
				},
			},
		},
	}

	aksCreateFull = &components.CreateClusterRequest{
		Name:             clusterRequestName,
		Location:         clusterRequestLocation,
		Cloud:            constants.Azure,
		NodeInstanceType: clusterRequestNodeInstance,
		SecretId:         clusterRequestSecretId,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy   `json:"dummy,omitempty"`
			CreateBYOC          *byoc.CreateBYOC            `json:"byoc,omitempty"`
		}{
			CreateClusterAzure: &azure.CreateClusterAzure{
				Node: &azure.CreateAzureNode{
					ResourceGroup:     clusterRequestRG,
					AgentCount:        clusterRequestNodeCount,
					AgentName:         clusterRequestAgentName,
					KubernetesVersion: clusterRequestKubernetes,
				},
			},
		},
	}

	aksEmptyLocationCreate = &components.CreateClusterRequest{
		Name:             clusterRequestName,
		Location:         "",
		Cloud:            constants.Azure,
		NodeInstanceType: clusterRequestNodeInstance,
		SecretId:         clusterRequestSecretId,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy   `json:"dummy,omitempty"`
			CreateBYOC          *byoc.CreateBYOC            `json:"byoc,omitempty"`
		}{
			CreateClusterAzure: &azure.CreateClusterAzure{
				Node: &azure.CreateAzureNode{
					ResourceGroup:     clusterRequestRG,
					AgentCount:        clusterRequestNodeCount,
					AgentName:         clusterRequestAgentName,
					KubernetesVersion: clusterRequestKubernetes,
				},
			},
		},
	}

	aksEmptyNITCreate = &components.CreateClusterRequest{
		Name:             clusterRequestName,
		Location:         clusterRequestLocation,
		Cloud:            constants.Azure,
		NodeInstanceType: "",
		SecretId:         clusterRequestSecretId,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy   `json:"dummy,omitempty"`
			CreateBYOC          *byoc.CreateBYOC            `json:"byoc,omitempty"`
		}{
			CreateClusterAzure: &azure.CreateClusterAzure{
				Node: &azure.CreateAzureNode{
					ResourceGroup:     clusterRequestRG,
					AgentCount:        clusterRequestNodeCount,
					AgentName:         clusterRequestAgentName,
					KubernetesVersion: clusterRequestKubernetes,
				},
			},
		},
	}

	awsCreateFull = &components.CreateClusterRequest{
		Name:             clusterRequestName,
		Location:         clusterRequestLocation,
		Cloud:            constants.Amazon,
		NodeInstanceType: clusterRequestNodeInstance,
		SecretId:         clusterRequestSecretId,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy   `json:"dummy,omitempty"`
			CreateBYOC          *byoc.CreateBYOC            `json:"byoc,omitempty"`
		}{
			CreateClusterAmazon: &amazon.CreateClusterAmazon{
				Node: &amazon.CreateAmazonNode{
					SpotPrice: clusterRequestSpotPrice,
					MinCount:  clusterRequestNodeCount,
					MaxCount:  clusterRequestNodeMaxCount,
					Image:     clusterRequestNodeImage,
				},
				Master: &amazon.CreateAmazonMaster{
					InstanceType: clusterRequestMasterInstance,
					Image:        clusterRequestMasterImage,
				},
			},
		},
	}

	dummyCreateFull = &components.CreateClusterRequest{
		Name:             clusterRequestName,
		Location:         clusterRequestLocation,
		Cloud:            constants.Dummy,
		NodeInstanceType: clusterRequestNodeInstance,
		SecretId:         clusterRequestSecretId,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy   `json:"dummy,omitempty"`
			CreateBYOC          *byoc.CreateBYOC            `json:"byoc,omitempty"`
		}{
			CreateClusterDummy: &dummy.CreateClusterDummy{
				Node: &dummy.Node{
					KubernetesVersion: clusterRequestKubernetes,
					Count:             clusterRequestNodeCount,
				},
			},
		},
	}

	awsEmptyLocationCreate = &components.CreateClusterRequest{
		Name:             clusterRequestName,
		Location:         "",
		Cloud:            constants.Amazon,
		NodeInstanceType: clusterRequestNodeInstance,
		SecretId:         clusterRequestSecretId,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy   `json:"dummy,omitempty"`
			CreateBYOC          *byoc.CreateBYOC            `json:"byoc,omitempty"`
		}{
			CreateClusterAmazon: &amazon.CreateClusterAmazon{
				Node: &amazon.CreateAmazonNode{
					SpotPrice: clusterRequestSpotPrice,
					MinCount:  clusterRequestNodeCount,
					MaxCount:  clusterRequestNodeMaxCount,
					Image:     clusterRequestNodeImage,
				},
				Master: &amazon.CreateAmazonMaster{
					InstanceType: clusterRequestMasterInstance,
					Image:        clusterRequestMasterImage,
				},
			},
		},
	}

	awsEmptyNITCreate = &components.CreateClusterRequest{
		Name:             clusterRequestName,
		Location:         clusterRequestLocation,
		Cloud:            constants.Amazon,
		NodeInstanceType: "",
		SecretId:         clusterRequestSecretId,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy   `json:"dummy,omitempty"`
			CreateBYOC          *byoc.CreateBYOC            `json:"byoc,omitempty"`
		}{
			CreateClusterAmazon: &amazon.CreateClusterAmazon{
				Node: &amazon.CreateAmazonNode{
					SpotPrice: clusterRequestSpotPrice,
					MinCount:  clusterRequestNodeCount,
					MaxCount:  clusterRequestNodeMaxCount,
					Image:     clusterRequestNodeImage,
				},
				Master: &amazon.CreateAmazonMaster{
					InstanceType: clusterRequestMasterInstance,
					Image:        clusterRequestMasterImage,
				},
			},
		},
	}

	byocCreateFull = &components.CreateClusterRequest{
		Name:             clusterRequestName,
		Location:         clusterRequestLocation,
		Cloud:            constants.BYOC,
		NodeInstanceType: clusterRequestNodeInstance,
		SecretId:         clusterRequestSecretId,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy   `json:"dummy,omitempty"`
			CreateBYOC          *byoc.CreateBYOC            `json:"byoc,omitempty"`
		}{
			CreateBYOC: &byoc.CreateBYOC{
				Metadata: map[string]string{
					clusterBYOCMetaKey: clusterBYOCMetaValue,
				},
			},
		},
	}

	byocEmptyLocationAndNIT = &components.CreateClusterRequest{
		Name:             clusterRequestName,
		Location:         "",
		Cloud:            constants.BYOC,
		NodeInstanceType: "",
		SecretId:         clusterRequestSecretId,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy   `json:"dummy,omitempty"`
			CreateBYOC          *byoc.CreateBYOC            `json:"byoc,omitempty"`
		}{
			CreateBYOC: &byoc.CreateBYOC{
				Metadata: map[string]string{
					clusterBYOCMetaKey: clusterBYOCMetaValue,
				},
			},
		},
	}

	notSupportedCloud = &components.CreateClusterRequest{
		Name:             clusterRequestName,
		Location:         clusterRequestLocation,
		Cloud:            "nonExistsCloud",
		NodeInstanceType: clusterRequestNodeInstance,
		SecretId:         clusterRequestSecretId,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy   `json:"dummy,omitempty"`
			CreateBYOC          *byoc.CreateBYOC            `json:"byoc,omitempty"`
		}{},
	}

	gkeWrongK8sVersion = &components.CreateClusterRequest{
		Name:             clusterRequestName,
		Location:         clusterRequestLocation,
		Cloud:            constants.Google,
		NodeInstanceType: clusterRequestNodeInstance,
		SecretId:         clusterRequestSecretId,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy   `json:"dummy,omitempty"`
			CreateBYOC          *byoc.CreateBYOC            `json:"byoc,omitempty"`
		}{
			CreateClusterGoogle: &google.CreateClusterGoogle{
				Project: clusterRequestProject,
				Node: &google.GoogleNode{
					Count:          clusterRequestNodeCount,
					Version:        clusterRequestWrongVersion,
					ServiceAccount: clusterServiceAccount,
				},
				Master: &google.GoogleMaster{
					Version: clusterRequestWrongVersion,
				},
			},
		},
	}

	gkeDifferentK8sVersion = &components.CreateClusterRequest{
		Name:             clusterRequestName,
		Location:         clusterRequestLocation,
		Cloud:            constants.Google,
		NodeInstanceType: clusterRequestNodeInstance,
		SecretId:         clusterRequestSecretId,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy   `json:"dummy,omitempty"`
			CreateBYOC          *byoc.CreateBYOC            `json:"byoc,omitempty"`
		}{
			CreateClusterGoogle: &google.CreateClusterGoogle{
				Project: clusterRequestProject,
				Node: &google.GoogleNode{
					Count:          clusterRequestNodeCount,
					Version:        clusterRequestVersion,
					ServiceAccount: clusterServiceAccount,
				},
				Master: &google.GoogleMaster{
					Version: clusterRequestVersion2,
				},
			},
		},
	}
)

var (
	gkeModelFull = &model.ClusterModel{
		Name:             clusterRequestName,
		Location:         clusterRequestLocation,
		NodeInstanceType: clusterRequestNodeInstance,
		SecretId:         clusterRequestSecretId,
		Cloud:            constants.Google,
		OrganizationId:   organizationId,
		Amazon:           model.AmazonClusterModel{},
		Azure:            model.AzureClusterModel{},
		Google: model.GoogleClusterModel{
			Project:        clusterRequestProject,
			MasterVersion:  clusterRequestVersion,
			NodeVersion:    clusterRequestVersion,
			NodeCount:      clusterRequestNodeCount,
			ServiceAccount: clusterServiceAccount,
		},
	}

	aksModelFull = &model.ClusterModel{
		Name:             clusterRequestName,
		Location:         clusterRequestLocation,
		NodeInstanceType: clusterRequestNodeInstance,
		SecretId:         clusterRequestSecretId,
		Cloud:            constants.Azure,
		OrganizationId:   organizationId,
		Amazon:           model.AmazonClusterModel{},
		Azure: model.AzureClusterModel{
			ResourceGroup:     clusterRequestRG,
			AgentCount:        clusterRequestNodeCount,
			AgentName:         clusterRequestAgentName,
			KubernetesVersion: clusterRequestKubernetes,
		},
		Google: model.GoogleClusterModel{},
	}

	awsModelFull = &model.ClusterModel{
		Name:             clusterRequestName,
		Location:         clusterRequestLocation,
		NodeInstanceType: clusterRequestNodeInstance,
		SecretId:         clusterRequestSecretId,
		Cloud:            constants.Amazon,
		OrganizationId:   organizationId,
		Amazon: model.AmazonClusterModel{
			NodeSpotPrice:      clusterRequestSpotPrice,
			NodeMinCount:       clusterRequestNodeCount,
			NodeMaxCount:       clusterRequestNodeMaxCount,
			NodeImage:          clusterRequestNodeImage,
			MasterInstanceType: clusterRequestMasterInstance,
			MasterImage:        clusterRequestMasterImage,
		},
		Azure:  model.AzureClusterModel{},
		Google: model.GoogleClusterModel{},
	}

	dummyModelFull = &model.ClusterModel{
		Name:             clusterRequestName,
		Location:         clusterRequestLocation,
		NodeInstanceType: clusterRequestNodeInstance,
		Cloud:            constants.Dummy,
		OrganizationId:   organizationId,
		SecretId:         clusterRequestSecretId,
		Amazon:           model.AmazonClusterModel{},
		Azure:            model.AzureClusterModel{},
		Google:           model.GoogleClusterModel{},
		Dummy: model.DummyClusterModel{
			KubernetesVersion: clusterRequestKubernetes,
			NodeCount:         clusterRequestNodeCount,
		},
	}

	byocModelFull = &model.ClusterModel{
		Name:             clusterRequestName,
		Location:         clusterRequestLocation,
		NodeInstanceType: clusterRequestNodeInstance,
		SecretId:         clusterRequestSecretId,
		Cloud:            constants.BYOC,
		OrganizationId:   organizationId,
		Amazon:           model.AmazonClusterModel{},
		Azure:            model.AzureClusterModel{},
		Google:           model.GoogleClusterModel{},
		BYOC: model.BYOClusterModel{
			Metadata: map[string]string{
				clusterBYOCMetaKey: clusterBYOCMetaValue,
			},
			MetadataRaw: nil,
		},
	}

	byocEmptyLocAndNIT = &model.ClusterModel{
		Name:             clusterRequestName,
		Location:         "",
		NodeInstanceType: "",
		SecretId:         clusterRequestSecretId,
		Cloud:            constants.BYOC,
		OrganizationId:   organizationId,
		Amazon:           model.AmazonClusterModel{},
		Azure:            model.AzureClusterModel{},
		Google:           model.GoogleClusterModel{},
		BYOC: model.BYOClusterModel{
			Metadata: map[string]string{
				clusterBYOCMetaKey: clusterBYOCMetaValue,
			},
			MetadataRaw: nil,
		},
	}

	gkeModelDifferentVersion = &model.ClusterModel{
		Name:             clusterRequestName,
		Location:         clusterRequestLocation,
		NodeInstanceType: clusterRequestNodeInstance,
		SecretId:         clusterRequestSecretId,
		Cloud:            constants.Google,
		OrganizationId:   organizationId,
		Amazon:           model.AmazonClusterModel{},
		Azure:            model.AzureClusterModel{},
		Google: model.GoogleClusterModel{
			Project:        clusterRequestProject,
			MasterVersion:  clusterRequestVersion2,
			NodeVersion:    clusterRequestVersion,
			NodeCount:      clusterRequestNodeCount,
			ServiceAccount: clusterServiceAccount,
		},
	}
)
