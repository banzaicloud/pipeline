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

package defaults_test

import (
	"reflect"
	"testing"

	"github.com/banzaicloud/pipeline/model/defaults"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/aks"
	"github.com/banzaicloud/pipeline/pkg/cluster/gke"
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
	}

	for _, tc := range testCases {

		t.Run(tc.name, func(t *testing.T) {
			err := tc.basicProfile.UpdateProfile(tc.request, false)

			if err != nil {
				t.Errorf("Expected error <nil>, got: %s", err.Error())
			}

			if !reflect.DeepEqual(tc.expectedResult, tc.basicProfile) {
				t.Errorf("Expected result: %#v, got: %#v", tc.expectedResult, tc.basicProfile)
			}

		})

	}

}

const (
	name             = "TestProfile"
	location         = "TestLocation"
	nodeInstanceType = "TestNodeInstance"
	version          = "TestVersion"
	nodeCount        = 1
	agentName        = "TestAgent"
	k8sVersion       = "TestKubernetesVersion"
)

// nolint: gochecknoglobals
var (
	fullRequestGKE = &pkgCluster.ClusterProfileRequest{
		Name:     name,
		Location: location,
		Cloud:    pkgCluster.Google,
		Properties: &pkgCluster.ClusterProfileProperties{
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
		Properties: &pkgCluster.ClusterProfileProperties{
			AKS: &aks.ClusterProfileAKS{
				KubernetesVersion: k8sVersion,
				NodePools: map[string]*aks.NodePoolCreate{
					agentName: {
						Count:            nodeCount,
						NodeInstanceType: nodeInstanceType,
						Labels:           map[string]string{"testname": "testvalue"},
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
				Labels: []*defaults.AKSNodePoolLabelsProfile{
					{
						Name:  "testname",
						Value: "testvalue",
					},
				},
			},
		},
	}
)

// nolint: gochecknoglobals
var (
	masterRequestGKE = &pkgCluster.ClusterProfileRequest{
		Name:     name,
		Location: location,
		Cloud:    pkgCluster.Google,
		Properties: &pkgCluster.ClusterProfileProperties{
			GKE: &gke.ClusterProfileGKE{
				Master: &gke.Master{
					Version: version,
				},
			},
		},
	}

	masterGKE = defaults.GKEProfile{
		DefaultModel:  defaults.DefaultModel{Name: name},
		Location:      location,
		MasterVersion: version,
	}
)

// nolint: gochecknoglobals
var (
	nodeRequestGKE = &pkgCluster.ClusterProfileRequest{
		Name:     name,
		Location: location,
		Cloud:    pkgCluster.Google,
		Properties: &pkgCluster.ClusterProfileProperties{
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
)

// nolint: gochecknoglobals
var (
	emptyRequestGKE = &pkgCluster.ClusterProfileRequest{
		Name:       name,
		Location:   location,
		Cloud:      pkgCluster.Google,
		Properties: &pkgCluster.ClusterProfileProperties{},
	}

	emptyRequestAKS = &pkgCluster.ClusterProfileRequest{
		Name:       name,
		Location:   location,
		Cloud:      pkgCluster.Azure,
		Properties: &pkgCluster.ClusterProfileProperties{},
	}

	emptyGKE = defaults.GKEProfile{
		DefaultModel: defaults.DefaultModel{Name: name},
		Location:     location,
	}

	emptyAKS = defaults.AKSProfile{
		DefaultModel: defaults.DefaultModel{Name: name},
		Location:     location,
	}
)
