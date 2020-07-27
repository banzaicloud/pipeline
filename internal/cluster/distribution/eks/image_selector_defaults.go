// Copyright © 2020 Banzai Cloud
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
	"github.com/Masterminds/semver/v3"
)

func mustConstraint(v string) *semver.Constraints {
	cs, err := semver.NewConstraint(fmt.Sprintf("~%s", v))
	if err != nil {
		emperror.Panic(errors.WrapIff(err, "could not create semver constraint for Kubernetes version %s.x", v))
	}

	return cs
}

// AMIs taken form https://docs.aws.amazon.com/eks/latest/userguide/eks-optimized-ami.html
// nolint: gochecknoglobals, deadcode
var defaultImages = ImageSelectors{
	KubernetesVersionImageSelector{ // Kubernetes Version 1.14.9
		Constraint: mustConstraint("1.14"),
		ImageSelector: RegionMapImageSelector{
			// AWS partition
			"ap-east-1":      "ami-03d6ba9854832e1af", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0472f72a6affbe2cc", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-01b6316fe22d918a9", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-069fad55139bcb636", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-0fbfeefbb99c1783d", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-09edbbb02478906e3", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-064fd810d76e41b31", // Canada (Central).
			"eu-central-1":   "ami-03d9393d97f5959fe", // EU (Frankfurt).
			"eu-north-1":     "ami-00aa667bc61a020ac", // EU (Stockholm).
			"eu-west-1":      "ami-048d37e92ce89022e", // EU (Ireland).
			"eu-west-2":      "ami-0a907f63b13a38029", // EU (London).
			"eu-west-3":      "ami-043b0c38ebd8435f9", // EU (Paris).
			"me-south-1":     "ami-0e337e92214d0764d", // Middle East (Bahrain).
			"sa-east-1":      "ami-0fb60915ec12aac26", // South America (Sao Paulo).
			"us-east-1":      "ami-05e621d4ba5b28dcc", // US East (N. Virginia).
			"us-east-2":      "ami-0b89776dcfa5f2dee", // US East (Ohio).
			"us-west-1":      "ami-0be2b482206616dc2", // US West (N. California).
			"us-west-2":      "ami-0a907f63b13a38029", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0d468258654a64de0", // GovCloud (US-East)
			"us-gov-west-1": "ami-0c82eb50449bdb498", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.15.11
		Constraint: mustConstraint("1.15"),
		ImageSelector: RegionMapImageSelector{
			"ap-east-1":      "ami-0ae1b0aad4d6fb508", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-026e39e61d44ff507", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0e1e660d5e393d5f1", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-0d32de2029a9f56fd", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-07aee5ce871a45bcf", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-040e5afd1b110a399", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-0286b3b75d600924d", // Canada (Central).
			"eu-central-1":   "ami-02497bca9c9dbc206", // EU (Frankfurt).
			"eu-north-1":     "ami-0efc0441f778f5826", // EU (Stockholm).
			"eu-west-1":      "ami-023736532608ff45e", // EU (Ireland).
			"eu-west-2":      "ami-0a79663bf395ae44d", // EU (London).
			"eu-west-3":      "ami-0de67e5d090c0eef0", // EU (Paris).
			"me-south-1":     "ami-075a74dc065b91bf6", // Middle East (Bahrain).
			"sa-east-1":      "ami-07b176b7f55c9df7c", // South America (Sao Paulo).
			"us-east-1":      "ami-06d4f570358b1b626", // US East (N. Virginia).
			"us-east-2":      "ami-0c1bd9eca9c869a0d", // US East (Ohio).
			"us-west-1":      "ami-03ac9a33b2d6c61f7", // US West (N. California).
			"us-west-2":      "ami-065418523a44331e5", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0a6bce000f1b3d6c3", // GovCloud (US-East)
			"us-gov-west-1": "ami-0ef1dfc4100d217ea", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.16.8
		Constraint: mustConstraint("1.16"),
		ImageSelector: RegionMapImageSelector{
			"ap-east-1":      "ami-024ec7439d902ac50", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0ca8e5c318b118092", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0da70348effa9af72", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-093fffbd1480f8c72", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-0ed6c012b2bd57b73", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-00725375bc29d9601", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-0139a5bcca4fd37a9", // Canada (Central).
			"eu-central-1":   "ami-0bf7306240d09dcdd", // EU (Frankfurt).
			"eu-north-1":     "ami-030f9071e0d32b989", // EU (Stockholm).
			"eu-west-1":      "ami-03c5e686287fd90e9", // EU (Ireland).
			"eu-west-2":      "ami-02e571568d897dcab", // EU (London).
			"eu-west-3":      "ami-0170bd57fa0c260da", // EU (Paris).
			"me-south-1":     "ami-08bf7870510670f47", // Middle East (Bahrain).
			"sa-east-1":      "ami-0ed2cc7e74c7bc2e2", // South America (Sao Paulo).
			"us-east-1":      "ami-05ac566a7ec2378db", // US East (N. Virginia).
			"us-east-2":      "ami-0edc51bc2f03c9dc2", // US East (Ohio).
			"us-west-1":      "ami-015fd1269b5bc0c23", // US West (N. California).
			"us-west-2":      "ami-0809659d79ce80260", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0dfbb8f14db9b6c28", // GovCloud (US-East)
			"us-gov-west-1": "ami-0849fbf34da6ce6cf", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.17.7
		Constraint: mustConstraint("1.17"),
		ImageSelector: RegionMapImageSelector{
			"ap-east-1":      "ami-0bb3febb3142599d2", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-06c747ba66b41a7ba", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-05ec1709521de4ece", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-0d536714e5639f906", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-04e1f56b76edf1adb", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0b9b4038f1f04c8d3", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-00ca7c831ef613b49", // Canada (Central).
			"eu-central-1":   "ami-0f0a5a541ea7e37e6", // EU (Frankfurt).
			"eu-north-1":     "ami-07bfdb27ea538e70e", // EU (Stockholm).
			"eu-west-1":      "ami-036a0d0e0582ceb89", // EU (Ireland).
			"eu-west-2":      "ami-0d556f731d5b55154", // EU (London).
			"eu-west-3":      "ami-07592d8accea4be97", // EU (Paris).
			"me-south-1":     "ami-0ae84a39f0799dcd7", // Middle East (Bahrain).
			"sa-east-1":      "ami-080e2d0ed87f73d34", // South America (Sao Paulo).
			"us-east-1":      "ami-07fb69750f0a547e7", // US East (N. Virginia).
			"us-east-2":      "ami-0f56d1834ab6cf42e", // US East (Ohio).
			"us-west-1":      "ami-083bfcb4c129eb7a7", // US West (N. California).
			"us-west-2":      "ami-085fc3ebd8fc486ff", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-048ccb979c3b25dfe", // GovCloud (US-East)
			"us-gov-west-1": "ami-05b1d6e2807536737", // GovCloud (US-West)
		},
	},
}

// DefaultImages returns an image selector that returns fallback images if no other images are found.
func DefaultImages() ImageSelector {
	return defaultImages
}
