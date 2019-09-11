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

package eks

import (
	"fmt"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	"github.com/Masterminds/semver"
	"github.com/aws/aws-sdk-go/aws/endpoints"
)

// ### [ Constants to EKS cluster default values ] ### //
const (
	DefaultInstanceType = "m4.xlarge"
	DefaultSpotPrice    = "0.0" // 0 spot price stands for on-demand instances
	DefaultRegion       = endpoints.UsWest2RegionID
	DefaultK8sVersion   = "1.14.6"
)

func constraintForVersion(v string) *semver.Constraints {
	cs, err := semver.NewConstraint(fmt.Sprintf("~%s", v))
	if err != nil {
		emperror.Panic(emperror.Wrap(err, fmt.Sprintf("could not create semver constraint for Kubernetes version %s.x", v)))
	}
	return cs
}

// AMIs taken form https://docs.aws.amazon.com/eks/latest/userguide/eks-optimized-ami.html
// nolint: gochecknoglobals
var mappings = []struct {
	constraint *semver.Constraints
	images     map[string]string
}{
	{
		constraintForVersion("1.11"),
		map[string]string{
			endpoints.UsEast2RegionID:      "ami-0f01aa60d08b56b51",
			endpoints.UsEast1RegionID:      "ami-0277d42c17d2fd9f6",
			endpoints.UsWest2RegionID:      "ami-0e024bc930f00f7a2",
			endpoints.ApEast1RegionID:      "ami-01efe02a8386b4448",
			endpoints.ApSouth1RegionID:     "ami-0fee6e6300019359b",
			endpoints.ApNortheast1RegionID: "ami-0acea8b886f0e3f8f",
			endpoints.ApNortheast2RegionID: "ami-01e754bb06dedfd13",
			endpoints.ApSoutheast1RegionID: "ami-04aff8301e51a47e4",
			endpoints.ApSoutheast2RegionID: "ami-0cb611c369549e9c9",
			endpoints.EuCentral1RegionID:   "ami-0157324517811932f",
			endpoints.EuWest1RegionID:      "ami-0ed6bcde59efbec8a",
			endpoints.EuWest2RegionID:      "ami-0b2d004b35f3153c4",
			endpoints.EuWest3RegionID:      "ami-0c592c8c15d5a2824",
			endpoints.EuNorth1RegionID:     "ami-019c885c71264830f",
			endpoints.MeSouth1RegionID:     "ami-0f19239ec6bfc1fd4",
		},
	},
	{
		constraintForVersion("1.12"),
		map[string]string{
			endpoints.UsEast2RegionID:      "ami-0d60b7264ed85e022",
			endpoints.UsEast1RegionID:      "ami-0259ce67309f76e0b",
			endpoints.UsWest2RegionID:      "ami-0315dd35bf204311d",
			endpoints.ApEast1RegionID:      "ami-0fafd65fe31195cb5",
			endpoints.ApSouth1RegionID:     "ami-0d9c7adc50f0c3f04",
			endpoints.ApNortheast1RegionID: "ami-08b2cecec9f2d5964",
			endpoints.ApNortheast2RegionID: "ami-0bbe543cd7fc2acd1",
			endpoints.ApSoutheast1RegionID: "ami-07696966feacc8e7b",
			endpoints.ApSoutheast2RegionID: "ami-07621fc1a7675f06c",
			endpoints.EuCentral1RegionID:   "ami-0fe22fc725c19301f",
			endpoints.EuWest1RegionID:      "ami-0a6be9528ebb8999d",
			endpoints.EuWest2RegionID:      "ami-0a8dc5b3290842d3e",
			endpoints.EuWest3RegionID:      "ami-01dbd2d713c939649",
			endpoints.EuNorth1RegionID:     "ami-0586fb63f5c466e3c",
			endpoints.MeSouth1RegionID:     "ami-08eab8b7cd9f43bd0",
		},
	},
	{
		constraintForVersion("1.13"),
		map[string]string{
			endpoints.UsEast2RegionID:      "ami-0355b5edf93d47112",
			endpoints.UsEast1RegionID:      "ami-08198f90fe8bc57f0",
			endpoints.UsWest2RegionID:      "ami-0dc5bf48daa40eb35",
			endpoints.ApEast1RegionID:      "ami-056314bd2d2acbdc1",
			endpoints.ApSouth1RegionID:     "ami-00f4cff050d28ee2d",
			endpoints.ApNortheast1RegionID: "ami-0262013b4d50142a2",
			endpoints.ApNortheast2RegionID: "ami-0d9a543e7c4279c11",
			endpoints.ApSoutheast1RegionID: "ami-0013f4890e2ce167b",
			endpoints.ApSoutheast2RegionID: "ami-01cd15b342b7edf5e",
			endpoints.EuCentral1RegionID:   "ami-01ffee931e45bb6bf",
			endpoints.EuWest1RegionID:      "ami-00ea6211202297fe8",
			endpoints.EuWest2RegionID:      "ami-0ef7099142dae7023",
			endpoints.EuWest3RegionID:      "ami-00cc28b5bcb9dc724",
			endpoints.EuNorth1RegionID:     "ami-01d7a7c38f882ef68",
			endpoints.MeSouth1RegionID:     "ami-0ae4a6a2950a3546e",
		},
	},
	{
		constraintForVersion("1.14"),
		map[string]string{
			endpoints.UsEast2RegionID:      "ami-0f841722be384ed96",
			endpoints.UsEast1RegionID:      "ami-08739803f18dcc019",
			endpoints.UsWest2RegionID:      "ami-038a987c6425a84ad",
			endpoints.ApEast1RegionID:      "ami-0fc4b0a16426993b5",
			endpoints.ApSouth1RegionID:     "ami-0e9f7f3edab94472d",
			endpoints.ApNortheast1RegionID: "ami-055d09694b6e5591a",
			endpoints.ApNortheast2RegionID: "ami-023bb403131889300",
			endpoints.ApSoutheast1RegionID: "ami-0d26e45a1e5422b8e",
			endpoints.ApSoutheast2RegionID: "ami-0d8e3da32bd74f39b",
			endpoints.EuCentral1RegionID:   "ami-0f64557dd6506a4aa",
			endpoints.EuWest1RegionID:      "ami-0497f6feb9d494baf",
			endpoints.EuWest2RegionID:      "ami-010d34c5744286662",
			endpoints.EuWest3RegionID:      "ami-04fada31d8c50b7a8",
			endpoints.EuNorth1RegionID:     "ami-0a4a5386eb62c775e",
			endpoints.MeSouth1RegionID:     "ami-0b7e753bbd3a0ae24",
		},
	},
}

// GetDefaultImageID returns the EKS optimized AMI for given Kubernetes version and region
func GetDefaultImageID(region, kubernetesVersion string) (string, error) {
	kubeVersion, err := semver.NewVersion(kubernetesVersion)
	if err != nil {
		return "", errors.WrapIfWithDetails(err, "could not create semver from Kubernetes version", "kubernetesVersion", kubernetesVersion)
	}

	for _, m := range mappings {
		if m.constraint.Check(kubeVersion) {
			if ami, ok := m.images[region]; ok {
				return ami, nil
			}

			return "", fmt.Errorf("no EKS AMI found for Kubernetes version %q in region %q", kubeVersion, region)
		}
	}

	return "", fmt.Errorf("unsupported Kubernetes version %q", kubeVersion)
}
