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
			"ap-east-1":      "ami-0ab30874529fd3e50", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-04bc347166d9e3aaf", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-04c89465bba8798a1", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-0755b21d0e95c0d0c", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-0709d8a66808430fb", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0ec2c3fadf19284f2", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-087df22a5ead73e75", // Canada (Central).
			"eu-central-1":   "ami-04c7280dc86f28713", // EU (Frankfurt).
			"eu-north-1":     "ami-0787d367983b3e798", // EU (Stockholm).
			"eu-west-1":      "ami-014bfbba28f19774e", // EU (Ireland).
			"eu-west-2":      "ami-064c1a4ee9bbfe23a", // EU (London).
			"eu-west-3":      "ami-0c2da9177c519bc2f", // EU (Paris).
			"me-south-1":     "ami-03559fb4f8a1c6f18", // Middle East (Bahrain).
			"sa-east-1":      "ami-0ecdd9c24a474a60a", // South America (Sao Paulo).
			"us-east-1":      "ami-0fef0f034f96ce511", // US East (N. Virginia).
			"us-east-2":      "ami-00b2135d346c29564", // US East (Ohio).
			"us-west-1":      "ami-0b8bf16f5f896c242", // US West (N. California).
			"us-west-2":      "ami-0d2c4583dc71806d6", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0d468258654a64de0", // GovCloud (US-East)
			"us-gov-west-1": "ami-0c82eb50449bdb498", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.15.11
		Constraint: mustConstraint("1.15"),
		ImageSelector: RegionMapImageSelector{
			"ap-east-1":      "ami-06c4a53520070412d", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0e263d94d831d6e3f", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0696aab7814e872d5", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-07ec30950a8cf5f5e", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-0163006b79677185b", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0fd5657d22dd97c7a", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-032072399f84866fa", // Canada (Central).
			"eu-central-1":   "ami-05acf7139b3fa4195", // EU (Frankfurt).
			"eu-north-1":     "ami-0cc9a5fbe0fb4846f", // EU (Stockholm).
			"eu-west-1":      "ami-0b4cbc24e98bbe268", // EU (Ireland).
			"eu-west-2":      "ami-051e5ec4ed42120bf", // EU (London).
			"eu-west-3":      "ami-0fced5d71992f332d", // EU (Paris).
			"me-south-1":     "ami-0b55422936e1febca", // Middle East (Bahrain).
			"sa-east-1":      "ami-0c882223525ac33e9", // South America (Sao Paulo).
			"us-east-1":      "ami-055e79c5dcb596625", // US East (N. Virginia).
			"us-east-2":      "ami-03c1ef6e2dcef9091", // US East (Ohio).
			"us-west-1":      "ami-07d781add09539e02", // US West (N. California).
			"us-west-2":      "ami-0b4f1df0761911a2a", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0a6bce000f1b3d6c3", // GovCloud (US-East)
			"us-gov-west-1": "ami-0ef1dfc4100d217ea", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.16.13
		Constraint: mustConstraint("1.16"),
		ImageSelector: RegionMapImageSelector{
			"ap-east-1":      "ami-005b3839f2d9dbb28", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-05db606f27c208dff", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-07a4a6b54bac7e1e5", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-055656b3b805b5875", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-04619364961f7cb8c", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-017cd1c6bec820e9d", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-06f9642a643dc1ef7", // Canada (Central).
			"eu-central-1":   "ami-0a2a6ee03ded5168d", // EU (Frankfurt).
			"eu-north-1":     "ami-0e342b3155c477ea2", // EU (Stockholm).
			"eu-west-1":      "ami-03156acdb42eb5a2b", // EU (Ireland).
			"eu-west-2":      "ami-03bf20b3bb5d00e90", // EU (London).
			"eu-west-3":      "ami-098270d0b239917a7", // EU (Paris).
			"me-south-1":     "ami-04a12d5f3b7983b34", // Middle East (Bahrain).
			"sa-east-1":      "ami-094d875bc33820805", // South America (Sao Paulo).
			"us-east-1":      "ami-0c8a11610abe0a666", // US East (N. Virginia).
			"us-east-2":      "ami-04cab20cf4ae39867", // US East (Ohio).
			"us-west-1":      "ami-077a11f5634192645", // US West (N. California).
			"us-west-2":      "ami-0841f061f8e44c4aa", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0dfbb8f14db9b6c28", // GovCloud (US-East)
			"us-gov-west-1": "ami-0849fbf34da6ce6cf", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.17.9
		Constraint: mustConstraint("1.17"),
		ImageSelector: RegionMapImageSelector{
			"ap-east-1":      "ami-092dc7701bd03af3e", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-095dcd341e28f2599", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-06a3ded9b6c463c6f", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-0365ad3c8cfd3cb4c", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-0e55c01aa1b1cf3cd", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0e51866c4b1e01c77", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-00a3415fa128f17c5", // Canada (Central).
			"eu-central-1":   "ami-0c28233a2bd46bd3e", // EU (Frankfurt).
			"eu-north-1":     "ami-043dbb11ff9b5a350", // EU (Stockholm).
			"eu-west-1":      "ami-0cb5f54d0d7b2ed21", // EU (Ireland).
			"eu-west-2":      "ami-05f8e36acad8edc61", // EU (London).
			"eu-west-3":      "ami-0d6f4cc928f18710e", // EU (Paris).
			"me-south-1":     "ami-0209971c7465bb090", // Middle East (Bahrain).
			"sa-east-1":      "ami-0ff3ff7ab99c06946", // South America (Sao Paulo).
			"us-east-1":      "ami-04125ecea1c9b3b27", // US East (N. Virginia).
			"us-east-2":      "ami-044cba456c7d6a2fe", // US East (Ohio).
			"us-west-1":      "ami-072fb5f7e7192bb7f", // US West (N. California).
			"us-west-2":      "ami-037843f6aeb12e236", // US West (Oregon).

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
