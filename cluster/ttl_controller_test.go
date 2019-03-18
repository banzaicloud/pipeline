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
	"reflect"
	"testing"
	"time"
)
import pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"

const ttl = 30 * time.Minute

// nolint: gochecknoglobals
var (
	clusterCreateTime         = time.Now().Add(-30 * time.Minute)
	clusterStartTimeWithinTTL = time.Now().Add(-20 * time.Minute)
	clusterStartTimeBeyondTTL = time.Now().Add(-40 * time.Minute)

	clusterCreating = &pkgCluster.GetClusterStatusResponse{
		Status: pkgCluster.Creating,
	}
	clusterRunning = &pkgCluster.GetClusterStatusResponse{
		CreatorBaseFields: *NewCreatorBaseFields(clusterCreateTime, 0),
		Status:            pkgCluster.Running,
		StartedAt:         &clusterStartTimeWithinTTL,
	}

	clusterRunningWithWarning = &pkgCluster.GetClusterStatusResponse{
		CreatorBaseFields: *NewCreatorBaseFields(clusterCreateTime, 0),
		Status:            pkgCluster.Running,
		StartedAt:         &clusterStartTimeWithinTTL,
	}

	oldClusterRunning = &pkgCluster.GetClusterStatusResponse{
		CreatorBaseFields: *NewCreatorBaseFields(clusterCreateTime, 0),
		Status:            pkgCluster.Running,
	}
)

func TestTtlController_isClusterEndOfLife(t *testing.T) {
	controller := NewTTLController(nil, nil, nil, nil)

	testCases := []struct {
		name            string
		clusterStarTime *time.Time
		ttl             time.Duration
		expected        bool
	}{
		{"cluster with no start time recorded", nil, ttl, false},
		{"running cluster within TTL", &clusterStartTimeWithinTTL, ttl, false},
		{"running cluster beyond TTL", &clusterStartTimeBeyondTTL, ttl, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := controller.isClusterEndOfLife(tc.clusterStarTime, tc.ttl)

			if actual != tc.expected {
				t.Errorf("isClusterEndOfLife expected return %v, got %v", tc.expected, actual)
			}
		})
	}
}

func TestTtlController_getClusterStartTime(t *testing.T) {
	controller := NewTTLController(nil, nil, nil, nil)

	testCases := []struct {
		name          string
		clusterDetail *pkgCluster.GetClusterStatusResponse
		expected      *time.Time
	}{
		{"no cluster detail available", nil, nil},
		{"cluster is creating", clusterCreating, nil},
		{"cluster is running", clusterRunning, &clusterStartTimeWithinTTL},
		{"cluster is running with warning", clusterRunningWithWarning, &clusterStartTimeWithinTTL},
		{"old cluster is running", oldClusterRunning, &clusterCreateTime},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := controller.getClusterStartTime(tc.clusterDetail)
			if !reflect.DeepEqual(tc.expected, actual) {
				t.Errorf("getClusterStartTime expected return %v, got %v", tc.expected, actual)
			}
		})
	}
}
