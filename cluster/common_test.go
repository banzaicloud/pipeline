package cluster_test

import (
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/components/amazon"
	"github.com/banzaicloud/banzai-types/components/azure"
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
	clusterRequestProject        = "testProject"
	clusterRequestNodeCount      = 1
	clusterRequestVersion        = "1.8.7-gke.1"
	clusterRequestVersion2       = "1.8.7-gke.2"
	clusterRequestWrongVersion   = "1.7.7-gke.1"
	clusterRequestRG             = "testResourceGroup"
	clusterRequestKubernetes     = "1.8.2"
	clusterRequestAgentName      = "testAgent"
	clusterRequestSpotPrice      = "1.2"
	clusterRequestNodeMaxCount   = 2
	clusterRequestNodeImage      = "testImage"
	clusterRequestMasterImage    = "testImage"
	clusterRequestMasterInstance = "testInstance"
	clusterServiceAccount        = "testServiceAccount"
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

		{name: "gke wrong k8s version", createRequest: gkeWrongK8sVersion, expectedModel: nil, expectedError: constants.ErrorWrongKubernetesVersion},
		{name: "gke different k8s version", createRequest: gkeDifferentK8sVersion, expectedModel: nil, expectedError: constants.ErrorDifferentKubernetesVersion},

		{name: "not supported cloud", createRequest: notSupportedCloud, expectedModel: nil, expectedError: constants.ErrorNotSupportedCloudType},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			commonCluster, err := cluster.CreateCommonClusterFromRequest(tc.createRequest)

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
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
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
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
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
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
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

	notSupportedCloud = &components.CreateClusterRequest{
		Name:             clusterRequestName,
		Location:         clusterRequestLocation,
		Cloud:            "nonExistsCloud",
		NodeInstanceType: clusterRequestNodeInstance,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
		}{},
	}

	gkeWrongK8sVersion = &components.CreateClusterRequest{
		Name:             clusterRequestName,
		Location:         clusterRequestLocation,
		Cloud:            constants.Google,
		NodeInstanceType: clusterRequestNodeInstance,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
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
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
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
		Cloud:            constants.Google,
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
		Cloud:            constants.Azure,
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
		Cloud:            constants.Amazon,
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
)
