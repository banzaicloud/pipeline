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

package cluster

import (
	"testing"
	"time"
)
import pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"

const ttl = 30

// nolint: gochecknoglobals
var (
	clusterCreating = &pkgCluster.StatusHistory{
		StatusChanges: []*pkgCluster.StatusChange{
			{
				CreatedAt:  time.Now().Add(-10 * time.Minute),
				FromStatus: pkgCluster.Unknown,
				ToStatus:   pkgCluster.Creating,
			},
		},
	}
	clusterRunningWithinTTL = &pkgCluster.StatusHistory{
		StatusChanges: []*pkgCluster.StatusChange{
			{
				CreatedAt:  time.Now().Add(-20 * time.Minute),
				FromStatus: pkgCluster.Unknown,
				ToStatus:   pkgCluster.Creating,
			},
			{
				CreatedAt:  time.Now().Add(-15 * time.Minute),
				FromStatus: pkgCluster.Creating,
				ToStatus:   pkgCluster.Running,
			},
			{
				CreatedAt:  time.Now().Add(-10 * time.Minute),
				FromStatus: pkgCluster.Running,
				ToStatus:   pkgCluster.Updating,
			},
			{
				CreatedAt:  time.Now().Add(-5 * time.Minute),
				FromStatus: pkgCluster.Updating,
				ToStatus:   pkgCluster.Warning,
			},
		},
	}

	clusterRunningBeyondTTL = &pkgCluster.StatusHistory{
		StatusChanges: []*pkgCluster.StatusChange{
			{
				CreatedAt:  time.Now().Add(-50 * time.Minute),
				FromStatus: pkgCluster.Unknown,
				ToStatus:   pkgCluster.Creating,
			},
			{
				CreatedAt:  time.Now().Add(-40 * time.Minute),
				FromStatus: pkgCluster.Creating,
				ToStatus:   pkgCluster.Running,
			},
			{
				CreatedAt:  time.Now().Add(-30 * time.Minute),
				FromStatus: pkgCluster.Running,
				ToStatus:   pkgCluster.Updating,
			},
			{
				CreatedAt:  time.Now().Add(-20 * time.Minute),
				FromStatus: pkgCluster.Updating,
				ToStatus:   pkgCluster.Running,
			},
		},
	}
)

func TestTtlController_isClusterEndOfLife(t *testing.T) {
	controller := NewTtlController(nil, nil, nil, nil)

	testCases := []struct {
		name          string
		statusHistory *pkgCluster.StatusHistory
		ttlMinutes    uint
		expected      bool
	}{
		{"cluster with no status history", nil, ttl, false},
		{"creating cluster", clusterCreating, ttl, false},
		{"running cluster within TTL", clusterRunningWithinTTL, ttl, false},
		{"running cluster beyond TTL", clusterRunningBeyondTTL, ttl, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := controller.isClusterEndOfLife(controller.getClusterStartTime(tc.statusHistory), tc.ttlMinutes)

			if actual != tc.expected {
				t.Errorf("isClusterEndOfLife expected return %v, got %v", tc.expected, actual)
			}
		})
	}

}
