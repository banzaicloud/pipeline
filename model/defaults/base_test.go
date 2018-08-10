package defaults_test

import (
	"testing"

	"github.com/banzaicloud/pipeline/model/defaults"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/acsk"
	"github.com/banzaicloud/pipeline/pkg/cluster/aks"
	"github.com/banzaicloud/pipeline/pkg/cluster/ec2"
	"github.com/banzaicloud/pipeline/pkg/cluster/eks"
	"github.com/banzaicloud/pipeline/pkg/cluster/gke"
	oracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/cluster"
	"github.com/banzaicloud/pipeline/utils"
)

func TestTableName(t *testing.T) {

	tableName := defaults.GKEProfile.TableName(defaults.GKEProfile{})
	if defaults.DefaultGKEProfileTableName != tableName {
		t.Errorf("Expected table name: %s, got: %s", defaults.DefaultGKEProfileTableName, tableName)
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
		{"type ec2", &defaults.EC2Profile{}, pkgCluster.Amazon}, // todo expand with other distribution
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			currentType := tc.profile.GetCloud()
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

		{"full request EC2", &defaults.EC2Profile{}, fullRequestEC2, &fullEC2},
		{"just master update EC2", &defaults.EC2Profile{}, masterRequestEC2, &masterEC2},
		{"just node update EC2", &defaults.EC2Profile{}, nodeRequestEC2, &nodeEC2},
		{"just basic update EC2", &defaults.EC2Profile{}, emptyRequestEC2, &emptyEC2}, // todo expand
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
			ACSK *acsk.ClusterProfileACSK `json:"acsk,omitempty"`
			EC2  *ec2.ClusterProfileEC2   `json:"ec2,omitempty"`
			EKS  *eks.ClusterProfileEKS   `json:"eks,omitempty"`
			AKS  *aks.ClusterProfileAKS   `json:"aks,omitempty"`
			GKE  *gke.ClusterProfileGKE   `json:"gke,omitempty"`
			OKE  *oracle.Cluster          `json:"oracle,omitempty"`
		}{
			GKE: &gke.ClusterProfileGKE{
				Master: &gke.Master{
					Version: version,
				},
				NodeVersion: version,
				NodePools: map[string]*gke.NodePool{
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
			ACSK *acsk.ClusterProfileACSK `json:"acsk,omitempty"`
			EC2  *ec2.ClusterProfileEC2   `json:"ec2,omitempty"`
			EKS  *eks.ClusterProfileEKS   `json:"eks,omitempty"`
			AKS  *aks.ClusterProfileAKS   `json:"aks,omitempty"`
			GKE  *gke.ClusterProfileGKE   `json:"gke,omitempty"`
			OKE  *oracle.Cluster          `json:"oracle,omitempty"`
		}{
			AKS: &aks.ClusterProfileAKS{
				KubernetesVersion: k8sVersion,
				NodePools: map[string]*aks.NodePoolCreate{
					agentName: {
						Count:            nodeCount,
						NodeInstanceType: nodeInstanceType,
					},
				},
			},
		},
	}

	fullRequestEC2 = &pkgCluster.ClusterProfileRequest{
		Name:     name,
		Location: location,
		Cloud:    pkgCluster.Amazon,
		Properties: struct {
			ACSK *acsk.ClusterProfileACSK `json:"acsk,omitempty"`
			EC2  *ec2.ClusterProfileEC2   `json:"ec2,omitempty"`
			EKS  *eks.ClusterProfileEKS   `json:"eks,omitempty"`
			AKS  *aks.ClusterProfileAKS   `json:"aks,omitempty"`
			GKE  *gke.ClusterProfileGKE   `json:"gke,omitempty"`
			OKE  *oracle.Cluster          `json:"oracle,omitempty"`
		}{
			EC2: &ec2.ClusterProfileEC2{
				Master: &ec2.ProfileMaster{
					InstanceType: masterInstanceType,
					Image:        masterImage,
				},
				NodePools: map[string]*ec2.NodePool{
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

	fullEC2 = defaults.EC2Profile{
		DefaultModel:       defaults.DefaultModel{Name: name},
		Location:           location,
		MasterInstanceType: masterInstanceType,
		MasterImage:        masterImage,
		NodePools: []*defaults.EC2NodePoolProfile{
			{
				AmazonNodePoolProfileBaseFields: defaults.AmazonNodePoolProfileBaseFields{
					InstanceType: nodeInstanceType,
					NodeName:     agentName,
					SpotPrice:    spotPrice,
					Autoscaling:  true,
					MinCount:     minCount,
					MaxCount:     maxCount,
					Count:        minCount,
				},
				Image: nodeImage,
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
			ACSK *acsk.ClusterProfileACSK `json:"acsk,omitempty"`
			EC2  *ec2.ClusterProfileEC2   `json:"ec2,omitempty"`
			EKS  *eks.ClusterProfileEKS   `json:"eks,omitempty"`
			AKS  *aks.ClusterProfileAKS   `json:"aks,omitempty"`
			GKE  *gke.ClusterProfileGKE   `json:"gke,omitempty"`
			OKE  *oracle.Cluster          `json:"oracle,omitempty"`
		}{
			GKE: &gke.ClusterProfileGKE{
				Master: &gke.Master{
					Version: version,
				},
			},
		},
	}

	masterRequestEC2 = &pkgCluster.ClusterProfileRequest{
		Name:     name,
		Location: location,
		Cloud:    pkgCluster.Amazon,
		Properties: struct {
			ACSK *acsk.ClusterProfileACSK `json:"acsk,omitempty"`
			EC2  *ec2.ClusterProfileEC2   `json:"ec2,omitempty"`
			EKS  *eks.ClusterProfileEKS   `json:"eks,omitempty"`
			AKS  *aks.ClusterProfileAKS   `json:"aks,omitempty"`
			GKE  *gke.ClusterProfileGKE   `json:"gke,omitempty"`
			OKE  *oracle.Cluster          `json:"oracle,omitempty"`
		}{
			EC2: &ec2.ClusterProfileEC2{
				Master: &ec2.ProfileMaster{
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

	masterEC2 = defaults.EC2Profile{
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
			ACSK *acsk.ClusterProfileACSK `json:"acsk,omitempty"`
			EC2  *ec2.ClusterProfileEC2   `json:"ec2,omitempty"`
			EKS  *eks.ClusterProfileEKS   `json:"eks,omitempty"`
			AKS  *aks.ClusterProfileAKS   `json:"aks,omitempty"`
			GKE  *gke.ClusterProfileGKE   `json:"gke,omitempty"`
			OKE  *oracle.Cluster          `json:"oracle,omitempty"`
		}{
			GKE: &gke.ClusterProfileGKE{
				NodeVersion: version,
				NodePools: map[string]*gke.NodePool{
					agentName: {
						Count:            nodeCount,
						NodeInstanceType: nodeInstanceType,
					},
				},
			},
		},
	}

	nodeRequestEC2 = &pkgCluster.ClusterProfileRequest{
		Name:     name,
		Location: location,
		Cloud:    pkgCluster.Amazon,
		Properties: struct {
			ACSK *acsk.ClusterProfileACSK `json:"acsk,omitempty"`
			EC2  *ec2.ClusterProfileEC2   `json:"ec2,omitempty"`
			EKS  *eks.ClusterProfileEKS   `json:"eks,omitempty"`
			AKS  *aks.ClusterProfileAKS   `json:"aks,omitempty"`
			GKE  *gke.ClusterProfileGKE   `json:"gke,omitempty"`
			OKE  *oracle.Cluster          `json:"oracle,omitempty"`
		}{
			EC2: &ec2.ClusterProfileEC2{
				NodePools: map[string]*ec2.NodePool{
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

	nodeEC2 = defaults.EC2Profile{
		DefaultModel: defaults.DefaultModel{Name: name},
		Location:     location,
		NodePools: []*defaults.EC2NodePoolProfile{
			{
				AmazonNodePoolProfileBaseFields: defaults.AmazonNodePoolProfileBaseFields{
					InstanceType: nodeInstanceType,
					NodeName:     agentName,
					SpotPrice:    spotPrice,
					Autoscaling:  true,
					MinCount:     minCount,
					MaxCount:     maxCount,
					Count:        minCount,
				},
				Image: nodeImage,
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

	emptyRequestEC2 = &pkgCluster.ClusterProfileRequest{
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

	emptyEC2 = defaults.EC2Profile{
		DefaultModel: defaults.DefaultModel{Name: name},
		Location:     location,
	}
)
