package defaults_test

import (
	"testing"

	"github.com/banzaicloud/pipeline/model/defaults"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/amazon"
	"github.com/banzaicloud/pipeline/pkg/cluster/azure"
	"github.com/banzaicloud/pipeline/pkg/cluster/eks"
	"github.com/banzaicloud/pipeline/pkg/cluster/google"
	oracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/cluster"
	"github.com/banzaicloud/pipeline/utils"
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
		{"type gke", &defaults.GKEProfile{}, pkgCluster.Google},
		{"type aks", &defaults.AKSProfile{}, pkgCluster.Azure},
		{"type aws", &defaults.AWSProfile{}, pkgCluster.Amazon},
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
		request        *pkgCluster.ClusterProfileRequest
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

const (
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
)

var (
	fullRequestGKE = &pkgCluster.ClusterProfileRequest{
		Name:     name,
		Location: location,
		Cloud:    pkgCluster.Google,
		Properties: struct {
			Amazon *amazon.ClusterProfileAmazon `json:"amazon,omitempty"`
			Azure  *azure.ClusterProfileAzure   `json:"azure,omitempty"`
			Eks    *eks.ClusterProfileEks       `json:"eks,omitempty"`
			Google *google.ClusterProfileGoogle `json:"google,omitempty"`
			Oracle *oracle.Cluster              `json:"oracle,omitempty"`
		}{
			Google: &google.ClusterProfileGoogle{
				Master: &google.Master{
					Version: version,
				},
				NodeVersion: version,
				NodePools: map[string]*google.NodePool{
					agentName: {
						Count:            nodeCount,
						NodeInstanceType: nodeInstanceType,
					},
				},
			},
		},
	}

	fullRequestAKS = &pkgCluster.ClusterProfileRequest{
		Name:     name,
		Location: location,
		Cloud:    pkgCluster.Azure,
		Properties: struct {
			Amazon *amazon.ClusterProfileAmazon `json:"amazon,omitempty"`
			Azure  *azure.ClusterProfileAzure   `json:"azure,omitempty"`
			Eks    *eks.ClusterProfileEks       `json:"eks,omitempty"`
			Google *google.ClusterProfileGoogle `json:"google,omitempty"`
			Oracle *oracle.Cluster              `json:"oracle,omitempty"`
		}{
			Azure: &azure.ClusterProfileAzure{
				KubernetesVersion: k8sVersion,
				NodePools: map[string]*azure.NodePoolCreate{
					agentName: {
						Count:            nodeCount,
						NodeInstanceType: nodeInstanceType,
					},
				},
			},
		},
	}

	fullRequestAWS = &pkgCluster.ClusterProfileRequest{
		Name:     name,
		Location: location,
		Cloud:    pkgCluster.Amazon,
		Properties: struct {
			Amazon *amazon.ClusterProfileAmazon `json:"amazon,omitempty"`
			Azure  *azure.ClusterProfileAzure   `json:"azure,omitempty"`
			Eks    *eks.ClusterProfileEks       `json:"eks,omitempty"`
			Google *google.ClusterProfileGoogle `json:"google,omitempty"`
			Oracle *oracle.Cluster              `json:"oracle,omitempty"`
		}{
			Amazon: &amazon.ClusterProfileAmazon{
				Master: &amazon.ProfileMaster{
					InstanceType: masterInstanceType,
					Image:        masterImage,
				},
				NodePools: map[string]*amazon.NodePool{
					agentName: {
						InstanceType: nodeInstanceType,
						SpotPrice:    spotPrice,
						Autoscaling:  true,
						Count:        minCount,
						MinCount:     minCount,
						MaxCount:     maxCount,
						Image:        nodeImage,
					},
				},
			},
		},
	}

	fullGKE = defaults.GKEProfile{
		DefaultModel:  defaults.DefaultModel{Name: name},
		Location:      location,
		NodeVersion:   version,
		MasterVersion: version,
		NodePools: []*defaults.GKENodePoolProfile{
			{
				Count:            nodeCount,
				NodeInstanceType: nodeInstanceType,
				NodeName:         agentName,
			},
		},
	}

	fullAKS = defaults.AKSProfile{
		DefaultModel:      defaults.DefaultModel{Name: name},
		Location:          location,
		KubernetesVersion: k8sVersion,
		NodePools: []*defaults.AKSNodePoolProfile{
			{
				NodeInstanceType: nodeInstanceType,
				Count:            nodeCount,
				NodeName:         agentName,
			},
		},
	}

	fullAWS = defaults.AWSProfile{
		DefaultModel:       defaults.DefaultModel{Name: name},
		Location:           location,
		MasterInstanceType: masterInstanceType,
		MasterImage:        masterImage,
		NodePools: []*defaults.AWSNodePoolProfile{
			{
				InstanceType: nodeInstanceType,
				NodeName:     agentName,
				SpotPrice:    spotPrice,
				Autoscaling:  true,
				MinCount:     minCount,
				MaxCount:     maxCount,
				Count:        minCount,
				Image:        nodeImage,
			},
		},
	}
)

var (
	masterRequestGKE = &pkgCluster.ClusterProfileRequest{
		Name:     name,
		Location: location,
		Cloud:    pkgCluster.Google,
		Properties: struct {
			Amazon *amazon.ClusterProfileAmazon `json:"amazon,omitempty"`
			Azure  *azure.ClusterProfileAzure   `json:"azure,omitempty"`
			Eks    *eks.ClusterProfileEks       `json:"eks,omitempty"`
			Google *google.ClusterProfileGoogle `json:"google,omitempty"`
			Oracle *oracle.Cluster              `json:"oracle,omitempty"`
		}{
			Google: &google.ClusterProfileGoogle{
				Master: &google.Master{
					Version: version,
				},
			},
		},
	}

	masterRequestAWS = &pkgCluster.ClusterProfileRequest{
		Name:     name,
		Location: location,
		Cloud:    pkgCluster.Amazon,
		Properties: struct {
			Amazon *amazon.ClusterProfileAmazon `json:"amazon,omitempty"`
			Azure  *azure.ClusterProfileAzure   `json:"azure,omitempty"`
			Eks    *eks.ClusterProfileEks       `json:"eks,omitempty"`
			Google *google.ClusterProfileGoogle `json:"google,omitempty"`
			Oracle *oracle.Cluster              `json:"oracle,omitempty"`
		}{
			Amazon: &amazon.ClusterProfileAmazon{
				Master: &amazon.ProfileMaster{
					InstanceType: masterInstanceType,
					Image:        masterImage,
				},
			},
		},
	}

	masterGKE = defaults.GKEProfile{
		DefaultModel:  defaults.DefaultModel{Name: name},
		Location:      location,
		MasterVersion: version,
	}

	masterAWS = defaults.AWSProfile{
		DefaultModel:       defaults.DefaultModel{Name: name},
		Location:           location,
		MasterInstanceType: masterInstanceType,
		MasterImage:        masterImage,
	}
)

var (
	nodeRequestGKE = &pkgCluster.ClusterProfileRequest{
		Name:     name,
		Location: location,
		Cloud:    pkgCluster.Google,
		Properties: struct {
			Amazon *amazon.ClusterProfileAmazon `json:"amazon,omitempty"`
			Azure  *azure.ClusterProfileAzure   `json:"azure,omitempty"`
			Eks    *eks.ClusterProfileEks       `json:"eks,omitempty"`
			Google *google.ClusterProfileGoogle `json:"google,omitempty"`
			Oracle *oracle.Cluster              `json:"oracle,omitempty"`
		}{
			Google: &google.ClusterProfileGoogle{
				NodeVersion: version,
				NodePools: map[string]*google.NodePool{
					agentName: {
						Count:            nodeCount,
						NodeInstanceType: nodeInstanceType,
					},
				},
			},
		},
	}

	nodeRequestAWS = &pkgCluster.ClusterProfileRequest{
		Name:     name,
		Location: location,
		Cloud:    pkgCluster.Amazon,
		Properties: struct {
			Amazon *amazon.ClusterProfileAmazon `json:"amazon,omitempty"`
			Azure  *azure.ClusterProfileAzure   `json:"azure,omitempty"`
			Eks    *eks.ClusterProfileEks       `json:"eks,omitempty"`
			Google *google.ClusterProfileGoogle `json:"google,omitempty"`
			Oracle *oracle.Cluster              `json:"oracle,omitempty"`
		}{
			Amazon: &amazon.ClusterProfileAmazon{
				NodePools: map[string]*amazon.NodePool{
					agentName: {
						InstanceType: nodeInstanceType,
						SpotPrice:    spotPrice,
						Autoscaling:  true,
						MinCount:     minCount,
						MaxCount:     maxCount,
						Count:        minCount,
						Image:        nodeImage,
					},
				},
			},
		},
	}

	nodeGKE = defaults.GKEProfile{
		DefaultModel: defaults.DefaultModel{Name: name},
		Location:     location,
		NodeVersion:  version,
		NodePools: []*defaults.GKENodePoolProfile{
			{
				Count:            nodeCount,
				NodeInstanceType: nodeInstanceType,
				NodeName:         agentName,
			},
		},
	}

	nodeAWS = defaults.AWSProfile{
		DefaultModel: defaults.DefaultModel{Name: name},
		Location:     location,
		NodePools: []*defaults.AWSNodePoolProfile{
			{
				InstanceType: nodeInstanceType,
				NodeName:     agentName,
				SpotPrice:    spotPrice,
				Autoscaling:  true,
				MinCount:     minCount,
				MaxCount:     maxCount,
				Count:        minCount,
				Image:        nodeImage,
			},
		},
	}
)

var (
	emptyRequestGKE = &pkgCluster.ClusterProfileRequest{
		Name:     name,
		Location: location,
		Cloud:    pkgCluster.Google,
	}

	emptyRequestAKS = &pkgCluster.ClusterProfileRequest{
		Name:     name,
		Location: location,
		Cloud:    pkgCluster.Azure,
	}

	emptyRequestAWS = &pkgCluster.ClusterProfileRequest{
		Name:     name,
		Location: location,
		Cloud:    pkgCluster.Amazon,
	}

	emptyGKE = defaults.GKEProfile{
		DefaultModel: defaults.DefaultModel{Name: name},
		Location:     location,
	}

	emptyAKS = defaults.AKSProfile{
		DefaultModel: defaults.DefaultModel{Name: name},
		Location:     location,
	}

	emptyAWS = defaults.AWSProfile{
		DefaultModel: defaults.DefaultModel{Name: name},
		Location:     location,
	}
)
