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

package adapter

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
)

func TestFillClusterFromClusterModel(t *testing.T) {
	cases := []struct {
		name     string
		input    cluster.ClusterModel
		expected pke.PKEOnAzureCluster
	}{
		{
			name:     "empty cluster model",
			input:    cluster.ClusterModel{},
			expected: pke.PKEOnAzureCluster{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var result pke.PKEOnAzureCluster
			fillClusterFromClusterModel(&result, tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
