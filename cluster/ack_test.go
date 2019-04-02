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

package cluster_test

import (
	"reflect"
	"testing"

	pipCluster "github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
)

// nolint: gochecknoglobals
var (
	id           = uint(99)
	idStr        = "myClusterId"
	name         = "myAlibabaCluster"
	location     = "eu-central-1a"
	region       = "eu-central-1"
	location2    = "eu-central-2a"
	region2      = "eu-central-2"
	cloud        = "alibaba"
	distribution = "ack"
	secretId     = "mySecretId"
	status       = "RUNNING"
	statusMsg    = "Cluster is running"

	aliCluster = &model.ClusterModel{
		ID:             id,
		Name:           name,
		Location:       location,
		Cloud:          cloud,
		Distribution:   distribution,
		OrganizationId: id,
		SecretId:       secretId,
		Status:         status,
		StatusMessage:  statusMsg,
		ACK: model.ACKClusterModel{
			ID:                id,
			ProviderClusterID: idStr,
			RegionID:          region,
			ZoneID:            location,
		},
	}

	aliCluster2 = &model.ClusterModel{
		ID:             id,
		Name:           name,
		Location:       location2,
		Cloud:          cloud,
		Distribution:   distribution,
		OrganizationId: id,
		SecretId:       secretId,
		Status:         status,
		StatusMessage:  statusMsg,
		ACK: model.ACKClusterModel{
			ID:                id,
			ProviderClusterID: idStr,
			RegionID:          region2,
			ZoneID:            location2,
		},
	}

	expectedStatus = &pkgCluster.GetClusterStatusResponse{
		Status:        status,
		StatusMessage: statusMsg,
		Name:          name,
		Location:      location,
		Cloud:         cloud,
		Distribution:  distribution,
		ResourceID:    id,
		Region:        region,
		NodePools:     map[string]*pkgCluster.NodePoolStatus{},
	}

	expectedStatus2 = &pkgCluster.GetClusterStatusResponse{
		Status:        status,
		StatusMessage: statusMsg,
		Name:          name,
		Location:      location2,
		Cloud:         cloud,
		Distribution:  distribution,
		ResourceID:    id,
		Region:        region2,
		NodePools:     map[string]*pkgCluster.NodePoolStatus{},
	}
)

func TestACKClusterStatus(t *testing.T) {

	testCases := []struct {
		name           string
		model          *model.ClusterModel
		expectedStatus *pkgCluster.GetClusterStatusResponse
	}{
		{name: "check Alibaba ACK status", model: aliCluster, expectedStatus: expectedStatus},
		{name: "check Alibaba ACK status 2", model: aliCluster2, expectedStatus: expectedStatus2},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cluster, err := pipCluster.CreateACKClusterFromModel(tc.model)
			if err != nil {
				t.Errorf("error during create ACK from model: %#v", err)
			} else {
				status, err := cluster.GetStatus()
				if err != nil {
					t.Errorf("error during getting cluster status: %#v", err)
				}

				if !reflect.DeepEqual(status, tc.expectedStatus) {
					t.Errorf("Expected model: %v, got: %v", tc.expectedStatus, status)
				}

			}

		})
	}

}
