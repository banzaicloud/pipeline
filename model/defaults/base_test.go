package defaults_test

import (
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/components/amazon"
	"github.com/banzaicloud/banzai-types/components/azure"
	"github.com/banzaicloud/banzai-types/components/google"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/model/defaults"
	"github.com/banzaicloud/pipeline/utils"
	"testing"
)

func TestTableName(t *testing.T) {

	tableName := defaults.GKEProfile.TableName(defaults.GKEProfile{})
	if defaults.DefaultGoogleProfileTablaName != tableName {
		t.Errorf("Expected table name: %s, got: %s", defaults.DefaultGoogleProfileTablaName, tableName)
	}

}

func TestGetType(t *testing.T) {

	cases := []struct {
		name         string
		profile      defaults.ClusterProfile
		expectedType string
	}{
		{"type gke", &defaults.GKEProfile{}, constants.Google},
		{"type aks", &defaults.AKSProfile{}, constants.Azure},
		{"type aws", &defaults.AWSProfile{}, constants.Amazon},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			currentType := tc.profile.GetType()
			if tc.expectedType != currentType {
				t.Errorf("Expected cloud type: %s, got: %s", tc.expectedType, currentType)
			}
		})
	}

}

func TestUpdateWithoutSave(t *testing.T) {

	testCases := []struct {
		name           string
		basicProfile   defaults.ClusterProfile
		request        *components.ClusterProfileRequest
		expectedResult defaults.ClusterProfile
	}{
		{"full request GKE", &defaults.GKEProfile{}, fullRequestGKE, &fullGKE},
		{"just master update GKE", &defaults.GKEProfile{}, masterRequestGKE, &masterGKE},
		{"just node update GKE", &defaults.GKEProfile{}, nodeRequestGKE, &nodeGKE},
		{"just basic update GKE", &defaults.GKEProfile{}, emptyRequestGKE, &emptyGKE},

		{"full request AKS", &defaults.AKSProfile{}, fullRequestAKS, &fullAKS},
		{"just basic update AKS", &defaults.AKSProfile{}, emptyRequestAKS, &emptyAKS},

		{"full request AWS", &defaults.AWSProfile{}, fullRequestAWS, &fullAWS},
		{"just master update AWS", &defaults.AWSProfile{}, masterRequestAWS, &masterAWS},
		{"just node update AWS", &defaults.AWSProfile{}, nodeRequestAWS, &nodeAWS},
		{"just basic update AWS", &defaults.AWSProfile{}, emptyRequestAWS, &emptyAWS},
	}

	for _, tc := range testCases {

		t.Run(tc.name, func(t *testing.T) {
			err := tc.basicProfile.UpdateProfile(tc.request, false)

			if err != nil {
				t.Errorf("Expected error <nil>, got: %s", err.Error())
			}

			if err := utils.IsDifferent(tc.expectedResult, tc.basicProfile); err == nil {
				t.Errorf("Expected result: %#v, got: %#v", tc.expectedResult, tc.basicProfile)
			}

		})

	}

}

var (
	name               = "TestProfile"
	location           = "TestLocation"
	nodeInstanceType   = "TestNodeInstance"
	masterInstanceType = "TestMasterInstance"
	masterImage        = "TestMasterImage"
	nodeImage          = "TestMasterImage"
	version            = "TestVersion"
	nodeCount          = 1
	agentName          = "TestAgent"
	k8sVersion         = "TestKubernetesVersion"
	minCount           = 1
	maxCount           = 2
	spotPrice          = "0.2"
	serviceAccount     = "TestServiceAccount"
)

var (
	fullRequestGKE = &components.ClusterProfileRequest{
		Name:             name,
		Location:         location,
		Cloud:            constants.Google,
		NodeInstanceType: nodeInstanceType,
		Properties: struct {
			Amazon *amazon.ClusterProfileAmazon `json:"amazon,omitempty"`
			Azure  *azure.ClusterProfileAzure   `json:"azure,omitempty"`
			Google *google.ClusterProfileGoogle `json:"google,omitempty"`
		}{
			Google: &google.ClusterProfileGoogle{
				Master: &google.GoogleMaster{
					Version: version,
				},
				Node: &google.GoogleNode{
					Count:          nodeCount,
					Version:        version,
					ServiceAccount: serviceAccount,
				},
			},
		},
	}

	fullRequestAKS = &components.ClusterProfileRequest{
		Name:             name,
		Location:         location,
		Cloud:            constants.Azure,
		NodeInstanceType: nodeInstanceType,
		Properties: struct {
			Amazon *amazon.ClusterProfileAmazon `json:"amazon,omitempty"`
			Azure  *azure.ClusterProfileAzure   `json:"azure,omitempty"`
			Google *google.ClusterProfileGoogle `json:"google,omitempty"`
		}{
			Azure: &azure.ClusterProfileAzure{
				Node: &azure.AzureProfileNode{
					AgentCount:        nodeCount,
					AgentName:         agentName,
					KubernetesVersion: k8sVersion,
				},
			},
		},
	}

	fullRequestAWS = &components.ClusterProfileRequest{
		Name:             name,
		Location:         location,
		Cloud:            constants.Amazon,
		NodeInstanceType: nodeInstanceType,
		Properties: struct {
			Amazon *amazon.ClusterProfileAmazon `json:"amazon,omitempty"`
			Azure  *azure.ClusterProfileAzure   `json:"azure,omitempty"`
			Google *google.ClusterProfileGoogle `json:"google,omitempty"`
		}{
			Amazon: &amazon.ClusterProfileAmazon{
				Master: &amazon.AmazonProfileMaster{
					InstanceType: masterInstanceType,
					Image:        masterImage,
				},
				Node: &amazon.AmazonProfileNode{
					SpotPrice: spotPrice,
					MinCount:  minCount,
					MaxCount:  maxCount,
					Image:     nodeImage,
				},
			},
		},
	}

	fullGKE = defaults.GKEProfile{
		DefaultModel:     defaults.DefaultModel{Name: name},
		Location:         location,
		NodeInstanceType: nodeInstanceType,
		MasterVersion:    version,
		NodeCount:        nodeCount,
		NodeVersion:      version,
		ServiceAccount:   serviceAccount,
	}

	fullAKS = defaults.AKSProfile{
		DefaultModel:      defaults.DefaultModel{Name: name},
		Location:          location,
		NodeInstanceType:  nodeInstanceType,
		AgentCount:        nodeCount,
		AgentName:         agentName,
		KubernetesVersion: k8sVersion,
	}

	fullAWS = defaults.AWSProfile{
		DefaultModel:       defaults.DefaultModel{Name: name},
		Location:           location,
		NodeInstanceType:   nodeInstanceType,
		NodeImage:          nodeImage,
		MasterInstanceType: masterInstanceType,
		MasterImage:        masterImage,
		NodeSpotPrice:      spotPrice,
		NodeMinCount:       minCount,
		NodeMaxCount:       maxCount,
	}
)

var (
	masterRequestGKE = &components.ClusterProfileRequest{
		Name:             name,
		Location:         location,
		Cloud:            constants.Google,
		NodeInstanceType: nodeInstanceType,
		Properties: struct {
			Amazon *amazon.ClusterProfileAmazon `json:"amazon,omitempty"`
			Azure  *azure.ClusterProfileAzure   `json:"azure,omitempty"`
			Google *google.ClusterProfileGoogle `json:"google,omitempty"`
		}{
			Google: &google.ClusterProfileGoogle{
				Master: &google.GoogleMaster{
					Version: version,
				},
			},
		},
	}

	masterRequestAWS = &components.ClusterProfileRequest{
		Name:             name,
		Location:         location,
		Cloud:            constants.Amazon,
		NodeInstanceType: nodeInstanceType,
		Properties: struct {
			Amazon *amazon.ClusterProfileAmazon `json:"amazon,omitempty"`
			Azure  *azure.ClusterProfileAzure   `json:"azure,omitempty"`
			Google *google.ClusterProfileGoogle `json:"google,omitempty"`
		}{
			Amazon: &amazon.ClusterProfileAmazon{
				Master: &amazon.AmazonProfileMaster{
					InstanceType: masterInstanceType,
					Image:        masterImage,
				},
			},
		},
	}

	masterGKE = defaults.GKEProfile{
		DefaultModel:     defaults.DefaultModel{Name: name},
		Location:         location,
		NodeInstanceType: nodeInstanceType,
		MasterVersion:    version,
	}

	masterAWS = defaults.AWSProfile{
		DefaultModel:       defaults.DefaultModel{Name: name},
		Location:           location,
		NodeInstanceType:   nodeInstanceType,
		MasterInstanceType: masterInstanceType,
		MasterImage:        masterImage,
	}
)

var (
	nodeRequestGKE = &components.ClusterProfileRequest{
		Name:             name,
		Location:         location,
		Cloud:            constants.Google,
		NodeInstanceType: nodeInstanceType,
		Properties: struct {
			Amazon *amazon.ClusterProfileAmazon `json:"amazon,omitempty"`
			Azure  *azure.ClusterProfileAzure   `json:"azure,omitempty"`
			Google *google.ClusterProfileGoogle `json:"google,omitempty"`
		}{
			Google: &google.ClusterProfileGoogle{
				Node: &google.GoogleNode{
					Count:   nodeCount,
					Version: version,
				},
			},
		},
	}

	nodeRequestAWS = &components.ClusterProfileRequest{
		Name:             name,
		Location:         location,
		Cloud:            constants.Amazon,
		NodeInstanceType: nodeInstanceType,
		Properties: struct {
			Amazon *amazon.ClusterProfileAmazon `json:"amazon,omitempty"`
			Azure  *azure.ClusterProfileAzure   `json:"azure,omitempty"`
			Google *google.ClusterProfileGoogle `json:"google,omitempty"`
		}{
			Amazon: &amazon.ClusterProfileAmazon{
				Node: &amazon.AmazonProfileNode{
					SpotPrice: spotPrice,
					MinCount:  minCount,
					MaxCount:  maxCount,
					Image:     nodeImage,
				},
			},
		},
	}

	nodeGKE = defaults.GKEProfile{
		DefaultModel:     defaults.DefaultModel{Name: name},
		Location:         location,
		NodeInstanceType: nodeInstanceType,
		NodeCount:        nodeCount,
		NodeVersion:      version,
	}

	nodeAWS = defaults.AWSProfile{
		DefaultModel:     defaults.DefaultModel{Name: name},
		Location:         location,
		NodeInstanceType: nodeInstanceType,
		NodeImage:        nodeImage,
		NodeSpotPrice:    spotPrice,
		NodeMinCount:     minCount,
		NodeMaxCount:     maxCount,
	}
)

var (
	emptyRequestGKE = &components.ClusterProfileRequest{
		Name:             name,
		Location:         location,
		Cloud:            constants.Google,
		NodeInstanceType: nodeInstanceType,
	}

	emptyRequestAKS = &components.ClusterProfileRequest{
		Name:             name,
		Location:         location,
		Cloud:            constants.Azure,
		NodeInstanceType: nodeInstanceType,
	}

	emptyRequestAWS = &components.ClusterProfileRequest{
		Name:             name,
		Location:         location,
		Cloud:            constants.Amazon,
		NodeInstanceType: nodeInstanceType,
	}

	emptyGKE = defaults.GKEProfile{
		DefaultModel:     defaults.DefaultModel{Name: name},
		Location:         location,
		NodeInstanceType: nodeInstanceType,
	}

	emptyAKS = defaults.AKSProfile{
		DefaultModel:     defaults.DefaultModel{Name: name},
		Location:         location,
		NodeInstanceType: nodeInstanceType,
	}

	emptyAWS = defaults.AWSProfile{
		DefaultModel:     defaults.DefaultModel{Name: name},
		Location:         location,
		NodeInstanceType: nodeInstanceType,
	}
)
