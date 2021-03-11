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
// nolint: gochecknoglobals
var defaultImages = ImageSelectors{
	KubernetesVersionImageSelector{ // Kubernetes Version 1.15.12
		Constraint: mustConstraint("1.15"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-0a22d931e879a5977", // Africa (Cape Town).
			"ap-east-1":      "ami-0966b7d985866d98b", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0b662cf0d2fe75ee5", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0199c4dbf5b6a2c74", // Asia Pacific (Seoul).
			"ap-northeast-3": "ami-0fcc25072fbcf1635", // Asia Pacific (Osaka)
			"ap-southeast-1": "ami-07a0af178adce4a92", // Asia Pacific (Singapore).
			"ap-southeast-2": "ami-0b200de0512771029", // Asia Pacific (Sydney).
			"ap-south-1":     "ami-0081c3f50b0696fa3", // Asia Pacific (Mumbai).
			"ca-central-1":   "ami-059fcce5f60766d63", // Canada (Central).
			"eu-central-1":   "ami-08d58cbe591275937", // EU (Frankfurt).
			"eu-north-1":     "ami-047fa06ef707e2ae6", // EU (Stockholm).
			"eu-south-1":     "ami-0d0af409c0cfc7b2a", // Europe (Milan).
			"eu-west-1":      "ami-06f86145e5cdeab92", // EU (Ireland).
			"eu-west-2":      "ami-0a7c03ebcdace09f2", // EU (London).
			"eu-west-3":      "ami-0c41191f398149147", // EU (Paris).
			"me-south-1":     "ami-084273bc658c69f76", // Middle East (Bahrain).
			"sa-east-1":      "ami-04117a3f33a3cd218", // South America (Sao Paulo).
			"us-east-1":      "ami-053db3d365fa3233d", // US East (N. Virginia).
			"us-east-2":      "ami-077e4ae424177ef68", // US East (Ohio).
			"us-west-1":      "ami-0db91ea9a81d29be4", // US West (N. California).
			"us-west-2":      "ami-0fb7648905bd3763e", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0fb0d8fe9da308372", // GovCloud (US-East)
			"us-gov-west-1": "ami-048bf426d8d2cf844", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.16.15
		Constraint: mustConstraint("1.16"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-019ef42adc029b87c", // Africa (Cape Town).
			"ap-east-1":      "ami-0d8158bf172db097a", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0b440db0908c1dc2d", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0953967e0addce4fd", // Asia Pacific (Seoul).
			"ap-northeast-3": "ami-07356ff47f883f75d", // Asia Pacific (Osaka)
			"ap-southeast-1": "ami-0493c7138e15ba133", // Asia Pacific (Singapore).
			"ap-southeast-2": "ami-04516c3b1235f2679", // Asia Pacific (Sydney).
			"ap-south-1":     "ami-03bfea50fd0da55f0", // Asia Pacific (Mumbai).
			"ca-central-1":   "ami-0d559c8ebe158b429", // Canada (Central).
			"eu-central-1":   "ami-049299f230d6bb636", // EU (Frankfurt).
			"eu-north-1":     "ami-039b33d2f258be756", // EU (Stockholm).
			"eu-south-1":     "ami-0cb6fcfce802030d7", // Europe (Milan).
			"eu-west-1":      "ami-0cab8ad6601dc4ebe", // EU (Ireland).
			"eu-west-2":      "ami-04d77db2e942f9589", // EU (London).
			"eu-west-3":      "ami-0d13b9211a36aa3ac", // EU (Paris).
			"me-south-1":     "ami-0dde1944b321b829b", // Middle East (Bahrain).
			"sa-east-1":      "ami-0f54148974d37d868", // South America (Sao Paulo).
			"us-east-1":      "ami-06949f8af797adb3f", // US East (N. Virginia).
			"us-east-2":      "ami-069d3b8c41ef73c6b", // US East (Ohio).
			"us-west-1":      "ami-03e1eaa2650f88192", // US West (N. California).
			"us-west-2":      "ami-07f404eae403dc0c6", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-00e242d6a1e229c7a", // GovCloud (US-East)
			"us-gov-west-1": "ami-08d4dbdabe9929ea8", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.17.12
		Constraint: mustConstraint("1.17"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-0004258add15e22a7", // Africa (Cape Town).
			"ap-east-1":      "ami-00df630e180444d19", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-041b387504f92c48f", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-032ee4bd64d42265b", // Asia Pacific (Seoul).
			"ap-northeast-3": "ami-04f6ded3154d026e3", // Asia Pacific (Osaka)
			"ap-southeast-1": "ami-06d623b43101e6591", // Asia Pacific (Singapore).
			"ap-southeast-2": "ami-02fca869dd4b9663d", // Asia Pacific (Sydney).
			"ap-south-1":     "ami-0478340655b60e4aa", // Asia Pacific (Mumbai).
			"ca-central-1":   "ami-02fb8729fb0a261da", // Canada (Central).
			"eu-central-1":   "ami-075bf179d56b04d36", // EU (Frankfurt).
			"eu-north-1":     "ami-0f96553c3ab17bd91", // EU (Stockholm).
			"eu-south-1":     "ami-0f1ae1f9e55e3ed0a", // Europe (Milan).
			"eu-west-1":      "ami-0dd0b8f4aef45e6c3", // EU (Ireland).
			"eu-west-2":      "ami-08ce4246bb693d3de", // EU (London).
			"eu-west-3":      "ami-0d9f042195635f8b9", // EU (Paris).
			"me-south-1":     "ami-0b948e14f8378963b", // Middle East (Bahrain).
			"sa-east-1":      "ami-041dd32a6a5082401", // South America (Sao Paulo).
			"us-east-1":      "ami-0738446cecce59ed3", // US East (N. Virginia).
			"us-east-2":      "ami-07cb461d6681b8059", // US East (Ohio).
			"us-west-1":      "ami-08c5fd490c48c983a", // US West (N. California).
			"us-west-2":      "ami-03f93716134347a0f", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-053a9c28655b04bea", // GovCloud (US-East)
			"us-gov-west-1": "ami-0749ee4df2e340a96", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.18.9
		Constraint: mustConstraint("1.18"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-00ac7c1ab791f57c6", // Africa (Cape Town).
			"ap-east-1":      "ami-07c03c0e1c51b5247", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0ded6a3b5b79ae72a", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-05c6df30e602c9e7b", // Asia Pacific (Seoul).
			"ap-northeast-3": "ami-07aabf7c8625e4517", // Asia Pacific (Osaka)
			"ap-southeast-1": "ami-0e27171db871f5800", // Asia Pacific (Singapore).
			"ap-southeast-2": "ami-03727269b97c37453", // Asia Pacific (Sydney).
			"ap-south-1":     "ami-059221fb4aa751c74", // Asia Pacific (Mumbai).
			"ca-central-1":   "ami-0a4828db42ecd5812", // Canada (Central).
			"eu-central-1":   "ami-02e17a76e494a9e99", // EU (Frankfurt).
			"eu-north-1":     "ami-07dbe03d8c9ed10c8", // EU (Stockholm).
			"eu-south-1":     "ami-029ea1c102c57d85f", // Europe (Milan).
			"eu-west-1":      "ami-0529ed3e27b91745e", // EU (Ireland).
			"eu-west-2":      "ami-084a7891ed928d99e", // EU (London).
			"eu-west-3":      "ami-09fcf2666f795b3cf", // EU (Paris).
			"me-south-1":     "ami-07f14671342d7244f", // Middle East (Bahrain).
			"sa-east-1":      "ami-043bf6b869d74acfe", // South America (Sao Paulo).
			"us-east-1":      "ami-067c876cd1b6fbcc4", // US East (N. Virginia).
			"us-east-2":      "ami-0920af85e97b8cc4f", // US East (Ohio).
			"us-west-1":      "ami-056f9eb2ce99a5ac7", // US West (N. California).
			"us-west-2":      "ami-0d2d9f8df7a5fc2a9", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0b61160fc50cd7001", // GovCloud (US-East)
			"us-gov-west-1": "ami-0ffca549706b8e692", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.19.6
		Constraint: mustConstraint("1.19"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-05a0e96498d3cedd1", // Africa (Cape Town).
			"ap-east-1":      "ami-09caada5b293192a0", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0bcaf6bd6bf21f59a", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0483083b3970fca57", // Asia Pacific (Seoul).
			"ap-northeast-3": "ami-06974895f827a52d1", // Asia Pacific (Osaka)
			"ap-southeast-1": "ami-01d55e1350f38909d", // Asia Pacific (Singapore).
			"ap-southeast-2": "ami-0e449349246caba58", // Asia Pacific (Sydney).
			"ap-south-1":     "ami-0d445556347d57490", // Asia Pacific (Mumbai).
			"ca-central-1":   "ami-04b15e94a9cefe78f", // Canada (Central).
			"eu-central-1":   "ami-07451a60c207d2516", // EU (Frankfurt).
			"eu-north-1":     "ami-0df502c9ff98451bf", // EU (Stockholm).
			"eu-south-1":     "ami-0148520a5ce5f4a0b", // Europe (Milan).
			"eu-west-1":      "ami-0b1f518179fbd9a6e", // EU (Ireland).
			"eu-west-2":      "ami-02d0618fa4623bf3b", // EU (London).
			"eu-west-3":      "ami-013e284adb8dbbf1f", // EU (Paris).
			"me-south-1":     "ami-0f5cdd508ad921397", // Middle East (Bahrain).
			"sa-east-1":      "ami-06fb74e43b669cb25", // South America (Sao Paulo).
			"us-east-1":      "ami-006432b755baeb1c6", // US East (N. Virginia).
			"us-east-2":      "ami-071fd4dd907408ea6", // US East (Ohio).
			"us-west-1":      "ami-0bdeadbb197e2806a", // US West (N. California).
			"us-west-2":      "ami-0d45bae218253b811", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-05c448c0c5d4919fa", // GovCloud (US-East)
			"us-gov-west-1": "ami-058971104981fbd4c", // GovCloud (US-West)
		},
	},
}

// DefaultImages returns an image selector that returns fallback images if no other images are found.
func DefaultImages() ImageSelector {
	return defaultImages
}

// GPU accelerated AMIs taken form https://docs.aws.amazon.com/eks/latest/userguide/eks-optimized-ami.html
// nolint: gochecknoglobals
var defaultAcceleratedImages = ImageSelectors{
	KubernetesVersionImageSelector{ // Kubernetes Version 1.15.12
		Constraint: mustConstraint("1.15"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-0d007b4e1bde5912b", // Africa (Cape Town).
			"ap-east-1":      "ami-062ca27594f3ff38b", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0569f01a7937396f9", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-04dbec5ad20a6feb9", // Asia Pacific (Seoul).
			"ap-northeast-3": "ami-0dd6687d43cf42916", // Asia Pacific (Osaka)
			"ap-southeast-1": "ami-03f6e3c0ed27a28ae", // Asia Pacific (Singapore).
			"ap-southeast-2": "ami-067f103d8dc06f33f", // Asia Pacific (Sydney).
			"ap-south-1":     "ami-0d3d2ec97a9dceeaa", // Asia Pacific (Mumbai).
			"ca-central-1":   "ami-0a8449aaaebc2e2df", // Canada (Central).
			"eu-central-1":   "ami-0fe272c3d76674556", // EU (Frankfurt).
			"eu-north-1":     "ami-0662a956035bbfb42", // EU (Stockholm).
			"eu-south-1":     "ami-0451efc69ac468dc2", // Europe (Milan).
			"eu-west-1":      "ami-0be85ef3b414f68c5", // EU (Ireland).
			"eu-west-2":      "ami-088da5e2cce6d91eb", // EU (London).
			"eu-west-3":      "ami-0d5998c07fcf411f4", // EU (Paris).
			"me-south-1":     "ami-0d82257004557a917", // Middle East (Bahrain).
			"sa-east-1":      "ami-08dec09e3b7054988", // South America (Sao Paulo).
			"us-east-1":      "ami-01162f308dbcb2807", // US East (N. Virginia).
			"us-east-2":      "ami-0d1b582de1a182607", // US East (Ohio).
			"us-west-1":      "ami-0179cc3786142268c", // US West (N. California).
			"us-west-2":      "ami-079375a348f2d88f5", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0e05f9685f98af72a", // GovCloud (US-East)
			"us-gov-west-1": "ami-0bb842a278ac0c8e1", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.16.15
		Constraint: mustConstraint("1.16"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-0290686fdae548e22", // Africa (Cape Town).
			"ap-east-1":      "ami-0d0cc36bbc61ea748", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-07d00b1330da6a740", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0c8562e66066f319c", // Asia Pacific (Seoul).
			"ap-northeast-3": "ami-06d84ca20a9bee38a", // Asia Pacific (Osaka)
			"ap-southeast-1": "ami-0d2b2874798d28bd5", // Asia Pacific (Singapore).
			"ap-southeast-2": "ami-0bf51a0ba581fb1e7", // Asia Pacific (Sydney).
			"ap-south-1":     "ami-0b521d147f2742a05", // Asia Pacific (Mumbai).
			"ca-central-1":   "ami-0e02e779301b5a8d6", // Canada (Central).
			"eu-central-1":   "ami-01eb2b9676333a1ab", // EU (Frankfurt).
			"eu-north-1":     "ami-04ff52638ac615d19", // EU (Stockholm).
			"eu-south-1":     "ami-04d272798944a32cc", // Europe (Milan).
			"eu-west-1":      "ami-059fc5fa7a5b68d46", // EU (Ireland).
			"eu-west-2":      "ami-06552fe660bdeb67b", // EU (London).
			"eu-west-3":      "ami-0352c7d5b3927789e", // EU (Paris).
			"me-south-1":     "ami-08940cbafd19c93fe", // Middle East (Bahrain).
			"sa-east-1":      "ami-0a0d39f82283668b3", // South America (Sao Paulo).
			"us-east-1":      "ami-020caee9f2fc31b4c", // US East (N. Virginia).
			"us-east-2":      "ami-03d47b8bde2cb86b9", // US East (Ohio).
			"us-west-1":      "ami-02aaddd8e965b6a8f", // US West (N. California).
			"us-west-2":      "ami-0bab6cbd335bee62e", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0789a54a1faa82ed7", // GovCloud (US-East)
			"us-gov-west-1": "ami-04a7703b3e112309f", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.17.12
		Constraint: mustConstraint("1.17"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-0f452c201fb4febdc", // Africa (Cape Town).
			"ap-east-1":      "ami-0d134503d0fc1fb55", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-02f07fe25d99f7406", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-09bd4c27dcbdf65be", // Asia Pacific (Seoul).
			"ap-northeast-3": "ami-0d97f6f02ee34ef6e", // Asia Pacific (Osaka)
			"ap-southeast-1": "ami-0acc036004be70324", // Asia Pacific (Singapore).
			"ap-southeast-2": "ami-018b18bb274876656", // Asia Pacific (Sydney).
			"ap-south-1":     "ami-07c82f96995e8d411", // Asia Pacific (Mumbai).
			"ca-central-1":   "ami-0a8c65f32c607d480", // Canada (Central).
			"eu-central-1":   "ami-0e8ec49268c9528d9", // EU (Frankfurt).
			"eu-north-1":     "ami-0b6948140ced7a2d5", // EU (Stockholm).
			"eu-south-1":     "ami-0f668e4978a0cb146", // Europe (Milan).
			"eu-west-1":      "ami-0aeab63df0a31ef07", // EU (Ireland).
			"eu-west-2":      "ami-0a5e7b0f6ed2c9ac6", // EU (London).
			"eu-west-3":      "ami-06b995235eaac80a6", // EU (Paris).
			"me-south-1":     "ami-093d27994ae0d61dd", // Middle East (Bahrain).
			"sa-east-1":      "ami-0e1ea0004fa1e90a7", // South America (Sao Paulo).
			"us-east-1":      "ami-050ae772871f6359c", // US East (N. Virginia).
			"us-east-2":      "ami-0694547ef4160fdba", // US East (Ohio).
			"us-west-1":      "ami-0a6f5f1a3ec04efff", // US West (N. California).
			"us-west-2":      "ami-0762945603e02c89d", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-080138bee36a9b9e7", // GovCloud (US-East)
			"us-gov-west-1": "ami-02cc2979e1deb94bb", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.18.9
		Constraint: mustConstraint("1.18"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-0d34a13aac03c5887", // Africa (Cape Town).
			"ap-east-1":      "ami-05f7648361bdfe757", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0bd016c8ece543c89", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0d5f5ac825d2490ce", // Asia Pacific (Seoul).
			"ap-northeast-3": "ami-0f2af7616493b7296", // Asia Pacific (Osaka)
			"ap-southeast-1": "ami-014e27cf829c53c0b", // Asia Pacific (Singapore).
			"ap-southeast-2": "ami-0503d03103b94e28c", // Asia Pacific (Sydney).
			"ap-south-1":     "ami-0e3ed658950ea624e", // Asia Pacific (Mumbai).
			"ca-central-1":   "ami-07db1b6f19641a26c", // Canada (Central).
			"eu-central-1":   "ami-0e687ffb374b67212", // EU (Frankfurt).
			"eu-north-1":     "ami-0541a089c57d3c7d3", // EU (Stockholm).
			"eu-south-1":     "ami-05a3f789a7c2b156a", // Europe (Milan).
			"eu-west-1":      "ami-0d04417f966900153", // EU (Ireland).
			"eu-west-2":      "ami-0b634a08e757d0b3a", // EU (London).
			"eu-west-3":      "ami-09c940e81068801d0", // EU (Paris).
			"me-south-1":     "ami-0151a7522ddcb6b4e", // Middle East (Bahrain).
			"sa-east-1":      "ami-017cf44f4b5198a59", // South America (Sao Paulo).
			"us-east-1":      "ami-0e8bbe15ba7136657", // US East (N. Virginia).
			"us-east-2":      "ami-00e314bcc8be8d70e", // US East (Ohio).
			"us-west-1":      "ami-078c5012cbf1e754a", // US West (N. California).
			"us-west-2":      "ami-07cb90e1bdc02f118", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-08540b575c713af56", // GovCloud (US-East)
			"us-gov-west-1": "ami-064058e89da66b2ea", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.19.6
		Constraint: mustConstraint("1.19"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-07aebccc8e2633625", // Africa (Cape Town).
			"ap-east-1":      "ami-0a03c7e5333510b1e", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-079178fd9ca61ed20", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-078b15719842f9db7", // Asia Pacific (Seoul).
			"ap-northeast-3": "ami-06ff86e3ccfcc3ca4", // Asia Pacific (Osaka)
			"ap-southeast-1": "ami-0694cdc67181ae9b4", // Asia Pacific (Singapore).
			"ap-southeast-2": "ami-01f073f2fed0b69f5", // Asia Pacific (Sydney).
			"ap-south-1":     "ami-0522171e6494ad953", // Asia Pacific (Mumbai).
			"ca-central-1":   "ami-08798c1f2ac4c9484", // Canada (Central).
			"eu-central-1":   "ami-0eedb97087500edcf", // EU (Frankfurt).
			"eu-north-1":     "ami-02d13e28cf46fb941", // EU (Stockholm).
			"eu-south-1":     "ami-04cd5ac9f427da459", // Europe (Milan).
			"eu-west-1":      "ami-0865725bf535184c5", // EU (Ireland).
			"eu-west-2":      "ami-042b3661a2a2bb831", // EU (London).
			"eu-west-3":      "ami-09412f26ca09c78a1", // EU (Paris).
			"me-south-1":     "ami-09a7a85eb27c80e2f", // Middle East (Bahrain).
			"sa-east-1":      "ami-0e4ca7e0afcfcf8f8", // South America (Sao Paulo).
			"us-east-1":      "ami-0a7a035cef97bbc43", // US East (N. Virginia).
			"us-east-2":      "ami-01ba4ff6d77ff84db", // US East (Ohio).
			"us-west-1":      "ami-0a9a741ad7ab673b2", // US West (N. California).
			"us-west-2":      "ami-025940c3146dc26c9", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0fb327ee4d5c91e96", // GovCloud (US-East)
			"us-gov-west-1": "ami-0dc270642bcf47354", // GovCloud (US-West)
		},
	},
}

// DefaultAcceleratedImages returns an image selector that returns fallback images if no other images are found.
func DefaultAcceleratedImages() ImageSelector {
	return defaultAcceleratedImages
}

// ARM architecture AMIs taken form https://docs.aws.amazon.com/eks/latest/userguide/eks-optimized-ami.html
// nolint: gochecknoglobals
var defaultARMImages = ImageSelectors{
	KubernetesVersionImageSelector{ // Kubernetes Version 1.15.12
		Constraint: mustConstraint("1.15"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-053f22b163127122a", // Africa (Cape Town).
			"ap-east-1":      "ami-000102215002485de", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-017fbdb6b0532b366", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0e07e55942790a584", // Asia Pacific (Seoul).
			"ap-northeast-3": "ami-051e46978a0aedb69", // Asia Pacific (Osaka)
			"ap-southeast-1": "ami-0dc986ece2ceb786b", // Asia Pacific (Singapore).
			"ap-southeast-2": "ami-0d200efb0407c0934", // Asia Pacific (Sydney).
			"ap-south-1":     "ami-0655dec31a9336874", // Asia Pacific (Mumbai).
			"ca-central-1":   "ami-0dbaf45e7f3d8bd9a", // Canada (Central).
			"eu-central-1":   "ami-0b38bc58811a5bcc6", // EU (Frankfurt).
			"eu-north-1":     "ami-07facde017a42902e", // EU (Stockholm).
			"eu-south-1":     "ami-03d57c6617da37cca", // Europe (Milan).
			"eu-west-1":      "ami-0ac2821f3b124e734", // EU (Ireland).
			"eu-west-2":      "ami-03f41a0c5f393210c", // EU (London).
			"eu-west-3":      "ami-0d17916b3bea3298c", // EU (Paris).
			"me-south-1":     "ami-00f6cbd79acc314ba", // Middle East (Bahrain).
			"sa-east-1":      "ami-0700527be6e689c2b", // South America (Sao Paulo).
			"us-east-1":      "ami-0b8810e80a50eab26", // US East (N. Virginia).
			"us-east-2":      "ami-05ec934431e4c8316", // US East (Ohio).
			"us-west-1":      "ami-04a4aecd631855517", // US West (N. California).
			"us-west-2":      "ami-085b980838e16310e", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0daeb759ac0f2dab5", // GovCloud (US-East)
			"us-gov-west-1": "ami-0536518a7f82bfc56", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.16.15
		Constraint: mustConstraint("1.16"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-07ad065a2185484a5", // Africa (Cape Town).
			"ap-east-1":      "ami-03bcc626ecedb75b3", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0a5113b84d853c12b", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0a2ce5654f01a0b85", // Asia Pacific (Seoul).
			"ap-northeast-3": "ami-038f1e2967f83f065", // Asia Pacific (Osaka)
			"ap-southeast-1": "ami-0dd98790b8449c0f6", // Asia Pacific (Singapore).
			"ap-southeast-2": "ami-0f6a6e8570d386353", // Asia Pacific (Sydney).
			"ap-south-1":     "ami-08e1b095f9423b547", // Asia Pacific (Mumbai).
			"ca-central-1":   "ami-0611b1ce2e4224d30", // Canada (Central).
			"eu-central-1":   "ami-0e5f7ce4b00a51aa0", // EU (Frankfurt).
			"eu-north-1":     "ami-02df902efcb00cd35", // EU (Stockholm).
			"eu-south-1":     "ami-06df67c497eadff5e", // Europe (Milan).
			"eu-west-1":      "ami-04c1c58d7d93331a8", // EU (Ireland).
			"eu-west-2":      "ami-00a91eb060d7fdede", // EU (London).
			"eu-west-3":      "ami-0204ac5ec5fd370ce", // EU (Paris).
			"me-south-1":     "ami-0bf437c87c4d5d90d", // Middle East (Bahrain).
			"sa-east-1":      "ami-029914d5cc2fac56e", // South America (Sao Paulo).
			"us-east-1":      "ami-0c7cd6691415e27f5", // US East (N. Virginia).
			"us-east-2":      "ami-052bc605257b58d7a", // US East (Ohio).
			"us-west-1":      "ami-0f8140a381cc7151d", // US West (N. California).
			"us-west-2":      "ami-0a416fc2dc2e40dbd", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0d5c7f0aef738c72d", // GovCloud (US-East)
			"us-gov-west-1": "ami-0d31641a6ca9e2f00", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.17.12
		Constraint: mustConstraint("1.17"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-00763aaf98995e842", // Africa (Cape Town).
			"ap-east-1":      "ami-038472625de9c12f9", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-066dadae898c1ef35", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0239734374821f79f", // Asia Pacific (Seoul).
			"ap-northeast-3": "ami-0235bedff131fa5f2", // Asia Pacific (Osaka)
			"ap-southeast-1": "ami-07e10a2f23699f128", // Asia Pacific (Singapore).
			"ap-southeast-2": "ami-09d3e5fdd771c4f5c", // Asia Pacific (Sydney).
			"ap-south-1":     "ami-0a9ff5f77c7a6db1e", // Asia Pacific (Mumbai).
			"ca-central-1":   "ami-05cde54834afbb756", // Canada (Central).
			"eu-central-1":   "ami-0eac33c4181063793", // EU (Frankfurt).
			"eu-north-1":     "ami-07827fed83e43dbed", // EU (Stockholm).
			"eu-south-1":     "ami-0b7119e77e896b3c5", // Europe (Milan).
			"eu-west-1":      "ami-0fb2b568e08dd8784", // EU (Ireland).
			"eu-west-2":      "ami-0cafe82dd28a54f11", // EU (London).
			"eu-west-3":      "ami-0a196734ea466fdd1", // EU (Paris).
			"me-south-1":     "ami-08cd3e72d4cc141e0", // Middle East (Bahrain).
			"sa-east-1":      "ami-064370964d0947237", // South America (Sao Paulo).
			"us-east-1":      "ami-04b48d1104968e165", // US East (N. Virginia).
			"us-east-2":      "ami-08f8f5209e3705652", // US East (Ohio).
			"us-west-1":      "ami-0be05335b67525542", // US West (N. California).
			"us-west-2":      "ami-08d3286cf05f77cac", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0c9fd0fe97e982b59", // GovCloud (US-East)
			"us-gov-west-1": "ami-0814a96f69a83b1f5", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.18.9
		Constraint: mustConstraint("1.18"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-024ce5b22f9a44d4a", // Africa (Cape Town).
			"ap-east-1":      "ami-0293925ac31fd276a", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-07ad852b6c663a669", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-01146bf83e193e945", // Asia Pacific (Seoul).
			"ap-northeast-3": "ami-02eddb4e5e5f5dc9f", // Asia Pacific (Osaka)
			"ap-southeast-1": "ami-02bbcb9c58aff9429", // Asia Pacific (Singapore).
			"ap-southeast-2": "ami-0fb420d86bdd31e8c", // Asia Pacific (Sydney).
			"ap-south-1":     "ami-0b513f42c1cadf91f", // Asia Pacific (Mumbai).
			"ca-central-1":   "ami-02bcbd851e5f11981", // Canada (Central).
			"eu-central-1":   "ami-06827eab55d88291a", // EU (Frankfurt).
			"eu-north-1":     "ami-02ef6362e5b3523fe", // EU (Stockholm).
			"eu-south-1":     "ami-07033565bf30e965d", // Europe (Milan).
			"eu-west-1":      "ami-03829394245add52c", // EU (Ireland).
			"eu-west-2":      "ami-05bb288cc6093d735", // EU (London).
			"eu-west-3":      "ami-077556d21ba63a494", // EU (Paris).
			"me-south-1":     "ami-0ecfb67a0a3b19cee", // Middle East (Bahrain).
			"sa-east-1":      "ami-0f81769fb42f40e6b", // South America (Sao Paulo).
			"us-east-1":      "ami-011105a4b180e45e7", // US East (N. Virginia).
			"us-east-2":      "ami-0e1c13582658bf8e4", // US East (Ohio).
			"us-west-1":      "ami-0be77e521a64df57d", // US West (N. California).
			"us-west-2":      "ami-0fbe5a35943c1a0f9", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-014120b28907d877f", // GovCloud (US-East)
			"us-gov-west-1": "ami-0845f427bd4be8831", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.19.6
		Constraint: mustConstraint("1.19"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-058661dfea9ddf8ff", // Africa (Cape Town).
			"ap-east-1":      "ami-08ac2eac0ddd470c3", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0a2858cb12b0634fe", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0dcd32a40af26a593", // Asia Pacific (Seoul).
			"ap-northeast-3": "ami-0e460e47795d1494a", // Asia Pacific (Osaka)
			"ap-southeast-1": "ami-08d8d21ead42bef4a", // Asia Pacific (Singapore).
			"ap-southeast-2": "ami-085717042f3d0f43f", // Asia Pacific (Sydney).
			"ap-south-1":     "ami-088e83dd1edb11e88", // Asia Pacific (Mumbai).
			"ca-central-1":   "ami-017d01eefa8d77946", // Canada (Central).
			"eu-central-1":   "ami-0ed742bbe9e765bcd", // EU (Frankfurt).
			"eu-north-1":     "ami-00a46ec5dab424506", // EU (Stockholm).
			"eu-south-1":     "ami-0ddf98fe6091eb3ec", // Europe (Milan).
			"eu-west-1":      "ami-0330d34ecf715c13e", // EU (Ireland).
			"eu-west-2":      "ami-0908a97d8ea946347", // EU (London).
			"eu-west-3":      "ami-04530fb3d061ae878", // EU (Paris).
			"me-south-1":     "ami-0e34f1bc38be738d4", // Middle East (Bahrain).
			"sa-east-1":      "ami-0daed92662f4558c1", // South America (Sao Paulo).
			"us-east-1":      "ami-0bb3cd37abd895c15", // US East (N. Virginia).
			"us-east-2":      "ami-0cda388c785c2a220", // US East (Ohio).
			"us-west-1":      "ami-058ce41ee3b9c1aa0", // US West (N. California).
			"us-west-2":      "ami-06cce67762666221c", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0736cd802d6a89bde", // GovCloud (US-East)
			"us-gov-west-1": "ami-05ef5576a80c7f1df", // GovCloud (US-West)
		},
	},
}

// DefaultARMImages returns an image selector that returns fallback images if no other images are found.
func DefaultARMImages() ImageSelector {
	return defaultARMImages
}
