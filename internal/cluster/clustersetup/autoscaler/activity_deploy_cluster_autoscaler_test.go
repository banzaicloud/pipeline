// Copyright © 2021 Banzai Cloud
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

package autoscaler

import (
	"errors"
	"testing"

	"go.uber.org/zap"

	"github.com/banzaicloud/pipeline/internal/global"
)

type K8sVersioner struct {
	k8sVersion string
}

func (k *K8sVersioner) GetKubernetesVersion() (string, error) {
	if k.k8sVersion != "" {
		return k.k8sVersion, nil
	}
	return "", errors.New("unknown version")
}

func TestGetImageVersion_Success(t *testing.T) {
	global.Config.Cluster.Autoscale.Charts.ClusterAutoscaler.ImageVersionConstraints = []struct {
		K8sVersion string
		Repository string
		Tag        string
	}{
		{
			K8sVersion: "<=1.12.x",
			Tag:        "v1.12.8",
			Repository: "gcr.io/google-containers/cluster-autoscaler",
		},
		{
			K8sVersion: "~1.13",
			Tag:        "v1.13.9",
			Repository: "gcr.io/google-containers/cluster-autoscaler",
		},
		{
			K8sVersion: ">=1.14",
			Tag:        "v1.14.7",
			Repository: "gcr.io/google-containers/cluster-autoscaler",
		},
	}

	testCases := []struct {
		name            string
		versioner       *K8sVersioner
		expectedVersion string
	}{
		{
			name:            "test matching version 1",
			versioner:       &K8sVersioner{k8sVersion: "1.12.2"},
			expectedVersion: "v1.12.8",
		},
		{
			name:            "test matching version 2",
			versioner:       &K8sVersioner{k8sVersion: "1.14.4"},
			expectedVersion: "v1.14.7",
		},
		{
			name:            "test no matching version 1",
			versioner:       &K8sVersioner{k8sVersion: "1.15.2"},
			expectedVersion: "v1.14.7",
		},
		{
			name:            "test no matching version 2",
			versioner:       &K8sVersioner{k8sVersion: "1.11"},
			expectedVersion: "v1.12.8",
		},
		{
			name:            "test unknown k8s version",
			versioner:       &K8sVersioner{},
			expectedVersion: "v1.14.7",
		},
	}

	logger := zap.NewNop().Sugar().With(
		"clusterID", 1,
		"workflowID", 1,
		"workflowRunID", 1,
	)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			version := getImageVersion(logger, tc.versioner)

			if version["tag"] != tc.expectedVersion {
				t.Errorf("Expected: %v, got: %v", tc.expectedVersion, version["tag"])
			}
		})
	}
}
