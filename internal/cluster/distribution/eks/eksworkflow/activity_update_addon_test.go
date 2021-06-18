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

package eksworkflow

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectLatestVersion(t *testing.T) {
	addonVersions := &eks.DescribeAddonVersionsOutput{
		Addons: []*eks.AddonInfo{
			{
				AddonName: aws.String("coredns"),
				AddonVersions: []*eks.AddonVersionInfo{
					{
						AddonVersion: aws.String("v1.7.0-eksbuild.1"),
						Compatibilities: []*eks.Compatibility{
							{
								ClusterVersion: aws.String("1.18"),
							},
						},
					},
					{
						AddonVersion: aws.String("v1.8.0-eksbuild.1"),
						Compatibilities: []*eks.Compatibility{
							{
								ClusterVersion: aws.String("1.18"),
							},
						},
					},
					{
						AddonVersion: aws.String("v1.8.3-eksbuild.1"),
						Compatibilities: []*eks.Compatibility{
							{
								ClusterVersion: aws.String("1.18"),
							},
						},
					},
					{
						AddonVersion: aws.String("v1.8.5-eksbuild.1"),
						Compatibilities: []*eks.Compatibility{
							{
								ClusterVersion: aws.String("1.19"),
							},
						},
					},
				},
			},
		},
	}

	t.Run("latest version is selected for 1.18", func(t *testing.T) {
		version, err := selectLatestVersion(addonVersions, "v1.7.0-eksbuild.1", "1.18")
		require.NoError(t, err)
		assert.Equal(t, "v1.8.3-eksbuild.1", version)
	})

	t.Run("latest version is selected for 1.19", func(t *testing.T) {
		version, err := selectLatestVersion(addonVersions, "v1.7.0-eksbuild.1", "1.19")
		require.NoError(t, err)
		assert.Equal(t, "v1.8.5-eksbuild.1", version)
	})

	t.Run("no available new version", func(t *testing.T) {
		version, err := selectLatestVersion(addonVersions, "v1.7.0-eksbuild.1", "1.20")
		require.NoError(t, err)
		assert.Equal(t, "v1.7.0-eksbuild.1", version)
	})
}
