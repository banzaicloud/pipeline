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

package ark

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigRequest(t *testing.T) {
	configRequest := ConfigRequest{
		Cluster: clusterConfig{
			Name:         "test",
			Provider:     "amazon",
			Distribution: "eks",
			Location:     "us-east-1",
			RBACEnabled:  false,
		},
		ClusterSecret: nil,
		Bucket: bucketConfig{
			Provider: "amazon",
			Name:     "testBucket",
			Prefix:   "test",
			Location: "us-east-1",
		},
		BucketSecret:     nil,
		UseClusterSecret: false,

		RestoreMode: false,
	}
	_, err := configRequest.getChartConfig()
	require.NoError(t, err)
}
