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
			// Kubernetes Version 1.11.10
			"ap-east-1":      "ami-01efe02a8386b4448", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0acea8b886f0e3f8f", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-01e754bb06dedfd13", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-04aff8301e51a47e4", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-0cb611c369549e9c9", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0fee6e6300019359b", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-011fae876f87ef80d", // Canada (Central).
			"eu-central-1":   "ami-0157324517811932f", // EU (Frankfurt).
			"eu-north-1":     "ami-019c885c71264830f", // EU (Stockholm).
			"eu-west-1":      "ami-0ed6bcde59efbec8a", // EU (Ireland).
			"eu-west-2":      "ami-0b2d004b35f3153c4", // EU (London).
			"eu-west-3":      "ami-0c592c8c15d5a2824", // EU (Paris).
			"me-south-1":     "ami-0f19239ec6bfc1fd4", // Middle East (Bahrain).
			"sa-east-1":      "ami-08258a284e4edb286", // South America (Sao Paulo).
			"us-east-1":      "ami-0277d42c17d2fd9f6", // US East (N. Virginia).
			"us-east-2":      "ami-0f01aa60d08b56b51", // US East (Ohio).
			"us-west-1":      "ami-0ea4bd52d952e37c2", // US West (N. California).
			"us-west-2":      "ami-0e024bc930f00f7a2", // US West (Oregon).
		},
	},
	{
		constraintForVersion("1.12"),
		map[string]string{
			// Kubernetes Version 1.12.10
			"ap-east-1":      "ami-0fafd65fe31195cb5", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-08b2cecec9f2d5964", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0bbe543cd7fc2acd1", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-07696966feacc8e7b", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-07621fc1a7675f06c", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0d9c7adc50f0c3f04", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-08c7289082f4c81f5", // Canada (Central).
			"eu-central-1":   "ami-0fe22fc725c19301f", // EU (Frankfurt).
			"eu-north-1":     "ami-0586fb63f5c466e3c", // EU (Stockholm).
			"eu-west-1":      "ami-0a6be9528ebb8999d", // EU (Ireland).
			"eu-west-2":      "ami-0a8dc5b3290842d3e", // EU (London).
			"eu-west-3":      "ami-01dbd2d713c939649", // EU (Paris).
			"me-south-1":     "ami-08eab8b7cd9f43bd0", // Middle East (Bahrain).
			"sa-east-1":      "ami-0426e4628be2eac14", // South America (Sao Paulo).
			"us-east-1":      "ami-0259ce67309f76e0b", // US East (N. Virginia).
			"us-east-2":      "ami-0d60b7264ed85e022", // US East (Ohio).
			"us-west-1":      "ami-0b9861f042f244ac8", // US West (N. California).
			"us-west-2":      "ami-0315dd35bf204311d", // US West (Oregon).
		},
	},
	{
		constraintForVersion("1.13"),
		map[string]string{
			// Kubernetes Version 1.13.10
			"ap-east-1":      "ami-056314bd2d2acbdc1", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0262013b4d50142a2", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0d9a543e7c4279c11", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-0013f4890e2ce167b", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-01cd15b342b7edf5e", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-00f4cff050d28ee2d", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-0c8695b5d2f053ad6", // Canada (Central).
			"eu-central-1":   "ami-01ffee931e45bb6bf", // EU (Frankfurt).
			"eu-north-1":     "ami-01d7a7c38f882ef68", // EU (Stockholm).
			"eu-west-1":      "ami-00ea6211202297fe8", // EU (Ireland).
			"eu-west-2":      "ami-0ef7099142dae7023", // EU (London).
			"eu-west-3":      "ami-00cc28b5bcb9dc724", // EU (Paris).
			"me-south-1":     "ami-0ae4a6a2950a3546e", // Middle East (Bahrain).
			"sa-east-1":      "ami-0fa3e4c30b6ef5414", // South America (Sao Paulo).
			"us-east-1":      "ami-08198f90fe8bc57f0", // US East (N. Virginia).
			"us-east-2":      "ami-0355b5edf93d47112", // US East (Ohio).
			"us-west-1":      "ami-0c646c8eee630aa1f", // US West (N. California).
			"us-west-2":      "ami-0dc5bf48daa40eb35", // US West (Oregon).
		},
	},
	{
		constraintForVersion("1.14"),
		map[string]string{
			// Kubernetes Version 1.14.6
			"ap-east-1":      "ami-0fc4b0a16426993b5", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-055d09694b6e5591a", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-023bb403131889300", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-0d26e45a1e5422b8e", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-0d8e3da32bd74f39b", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0e9f7f3edab94472d", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-078e7f0499f9e8860", // Canada (Central).
			"eu-central-1":   "ami-0f64557dd6506a4aa", // EU (Frankfurt).
			"eu-north-1":     "ami-0a4a5386eb62c775e", // EU (Stockholm).
			"eu-west-1":      "ami-0497f6feb9d494baf", // EU (Ireland).
			"eu-west-2":      "ami-010d34c5744286662", // EU (London).
			"eu-west-3":      "ami-04fada31d8c50b7a8", // EU (Paris).
			"me-south-1":     "ami-0b7e753bbd3a0ae24", // Middle East (Bahrain).
			"sa-east-1":      "ami-0af97d6bb25294265", // South America (Sao Paulo).
			"us-east-1":      "ami-08739803f18dcc019", // US East (N. Virginia).
			"us-east-2":      "ami-0f841722be384ed96", // US East (Ohio).
			"us-west-1":      "ami-0d1096db9016114f9", // US West (N. California).
			"us-west-2":      "ami-038a987c6425a84ad", // US West (Oregon).
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
