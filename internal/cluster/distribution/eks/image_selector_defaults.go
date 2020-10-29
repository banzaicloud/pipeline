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
	KubernetesVersionImageSelector{ // Kubernetes Version 1.14.9
		Constraint: mustConstraint("1.14"),
		ImageSelector: RegionMapImageSelector{
			// AWS partition
			"af-south-1":     "ami-0f8ab65580bd719d2", // Africa (Cape Town).
			"ap-east-1":      "ami-0f0f4029b62a5b7ad", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-002cc7fc4eb2a75e4", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-06d29731aa23ea686", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-002011557064786e9", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-00f215232f5667d89", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0f6d6ab99f6982f1d", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-08b3ecd5f94947805", // Canada (Central).
			"eu-central-1":   "ami-094a5a8570e5ddae3", // EU (Frankfurt).
			"eu-north-1":     "ami-01009f85b2ccaf4e2", // EU (Stockholm).
			"eu-south-1":     "ami-0eacd9f01f2d144bd", // Europe (Milan).
			"eu-west-1":      "ami-0765701b78242bdc6", // EU (Ireland).
			"eu-west-2":      "ami-03125894801e890af", // EU (London).
			"eu-west-3":      "ami-07f68a0967f158d8e", // EU (Paris).
			"me-south-1":     "ami-0e7c79eed3d661e90", // Middle East (Bahrain).
			"sa-east-1":      "ami-0a68d783d2d36714f", // South America (Sao Paulo).
			"us-east-1":      "ami-02eb17dca4560eaa9", // US East (N. Virginia).
			"us-east-2":      "ami-0fe772897b653f8e6", // US East (Ohio).
			"us-west-1":      "ami-0e6ecdd32fad4b48a", // US West (N. California).
			"us-west-2":      "ami-0e90a4bb48c27d4df", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0d468258654a64de0", // GovCloud (US-East)
			"us-gov-west-1": "ami-0c82eb50449bdb498", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.15.11
		Constraint: mustConstraint("1.15"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-0578b807b719cabdd", // Africa (Cape Town).
			"ap-east-1":      "ami-034382e3d11643778", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-05d5811f019627d23", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0693d32b5c575531b", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-0eeabedc8395abe2e", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-07f5c294575341e65", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0e8b410d739253a36", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-0572f3b71cdaa6b3f", // Canada (Central).
			"eu-central-1":   "ami-0503f8258d7df6001", // EU (Frankfurt).
			"eu-north-1":     "ami-0639d11d35fe64d2c", // EU (Stockholm).
			"eu-south-1":     "ami-0ef9a055d02891de6", // Europe (Milan).
			"eu-west-1":      "ami-05560ab1b38de02bb", // EU (Ireland).
			"eu-west-2":      "ami-0c715d71bded1c40e", // EU (London).
			"eu-west-3":      "ami-092783c696a6db223", // EU (Paris).
			"me-south-1":     "ami-0f7602db7a6b89cb2", // Middle East (Bahrain).
			"sa-east-1":      "ami-06488a769fa5f75c8", // South America (Sao Paulo).
			"us-east-1":      "ami-02bab75c1a52eba68", // US East (N. Virginia).
			"us-east-2":      "ami-0d38d43582b0a32bf", // US East (Ohio).
			"us-west-1":      "ami-05851f70754547d8d", // US West (N. California).
			"us-west-2":      "ami-056b54ee08fc2e5d1", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0a6bce000f1b3d6c3", // GovCloud (US-East)
			"us-gov-west-1": "ami-0ef1dfc4100d217ea", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.16.13
		Constraint: mustConstraint("1.16"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-074166677c8aa1d39", // Africa (Cape Town).
			"ap-east-1":      "ami-054ab980feb464a27", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0b6f41e05739de6f7", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0c4fddcab827ce47e", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-08e29804ac624d84c", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-011d40a39322ee07a", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-05618375e5494e025", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-0b5716c37f90ebd4d", // Canada (Central).
			"eu-central-1":   "ami-00341e507eb458a09", // EU (Frankfurt).
			"eu-north-1":     "ami-08f8bd5d5365ef386", // EU (Stockholm).
			"eu-south-1":     "ami-0a7dab74109ffc60e", // Europe (Milan).
			"eu-west-1":      "ami-0c62031215595b87a", // EU (Ireland).
			"eu-west-2":      "ami-060c87ac45f6e8e1f", // EU (London).
			"eu-west-3":      "ami-08d3272747bd24ad7", // EU (Paris).
			"me-south-1":     "ami-0080a7be6ffe76791", // Middle East (Bahrain).
			"sa-east-1":      "ami-01f36e4946e9683aa", // South America (Sao Paulo).
			"us-east-1":      "ami-04e4992e477024f96", // US East (N. Virginia).
			"us-east-2":      "ami-0fbc7e56fb99f5337", // US East (Ohio).
			"us-west-1":      "ami-0eef2afe1fce7c4c4", // US West (N. California).
			"us-west-2":      "ami-0412dd0339679edc9", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0dfbb8f14db9b6c28", // GovCloud (US-East)
			"us-gov-west-1": "ami-0849fbf34da6ce6cf", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.17.9
		Constraint: mustConstraint("1.17"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-0f9cc438e5b3bf53e", // Africa (Cape Town).
			"ap-east-1":      "ami-0640011b3ac49d33f", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-048669b0687eb3ad4", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-051a4e2ffdcf3ec03", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-08468dfdc5c74b9b6", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-0e86ebda4a10a0c9b", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0f23e10e68fbfad61", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-0a1c0fe138d36c907", // Canada (Central).
			"eu-central-1":   "ami-065ebfccbbb1c5734", // EU (Frankfurt).
			"eu-north-1":     "ami-03b82a6422a5fea81", // EU (Stockholm).
			"eu-south-1":     "ami-090d32cb702e21337", // Europe (Milan).
			"eu-west-1":      "ami-0e88c067f159fff29", // EU (Ireland).
			"eu-west-2":      "ami-0a0e935731c835095", // EU (London).
			"eu-west-3":      "ami-08ace09f7833fbcd4", // EU (Paris).
			"me-south-1":     "ami-0c18449eb960939bb", // Middle East (Bahrain).
			"sa-east-1":      "ami-057a7809330139961", // South America (Sao Paulo).
			"us-east-1":      "ami-0925e0a4a64fb6895", // US East (N. Virginia).
			"us-east-2":      "ami-072868043b527ff26", // US East (Ohio).
			"us-west-1":      "ami-083b873a7c9822ea0", // US West (N. California).
			"us-west-2":      "ami-01c3b376c82d7673d", // US West (Oregon).

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

// GPU accelerated AMIs taken form https://docs.aws.amazon.com/eks/latest/userguide/eks-optimized-ami.html
// nolint: gochecknoglobals
var defaultAcceleratedImages = ImageSelectors{
	KubernetesVersionImageSelector{ // Kubernetes Version 1.14.9
		Constraint: mustConstraint("1.14"),
		ImageSelector: RegionMapImageSelector{
			// AWS partition
			"af-south-1":     "ami-06b33f1fcb42c30f0", // Africa (Cape Town).
			"ap-east-1":      "ami-0b8c96df50453c6fd", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-01f0ad047b7af5829", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0089d082d47798ca1", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-0eea784352bf9b337", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-0ace581a6da5e617b", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0fa8f519f193c98da", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-08a8fe5f6f92410b4", // Canada (Central).
			"eu-central-1":   "ami-0602b74a758f03334", // EU (Frankfurt).
			"eu-north-1":     "ami-051253b40b25c8cfb", // EU (Stockholm).
			"eu-south-1":     "ami-08a0e1c5cdacd5dcd", // Europe (Milan).
			"eu-west-1":      "ami-0422c11b98004da78", // EU (Ireland).
			"eu-west-2":      "ami-0454d3f8a3887dc39", // EU (London).
			"eu-west-3":      "ami-040457a3ec13e689b", // EU (Paris).
			"me-south-1":     "ami-00aef9060ea006deb", // Middle East (Bahrain).
			"sa-east-1":      "ami-0c07ec6859a38a390", // South America (Sao Paulo).
			"us-east-1":      "ami-05c9e2f3a47d026d3", // US East (N. Virginia).
			"us-east-2":      "ami-04cbbb18cedbf46e7", // US East (Ohio).
			"us-west-1":      "ami-03d3501bb755e618a", // US West (N. California).
			"us-west-2":      "ami-0963fa5ab98a1ae78", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0f20481af98657a3f", // GovCloud (US-East)
			"us-gov-west-1": "ami-05e8032c7a9091ed8", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.15.11
		Constraint: mustConstraint("1.15"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-01e400dbc00003039", // Africa (Cape Town).
			"ap-east-1":      "ami-03b83a1d4c03acbf1", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0fb2fcc2ebea66788", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0297c509fe7d56135", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-0572bbd844ee4dc36", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-0bce3715ecf995cbc", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-09835fe28827a0185", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-0e2679630c770c68b", // Canada (Central).
			"eu-central-1":   "ami-0d5041a9735ca2789", // EU (Frankfurt).
			"eu-north-1":     "ami-04fc4e52561e4c6c7", // EU (Stockholm).
			"eu-south-1":     "ami-0f81db845d6ae473c", // Europe (Milan).
			"eu-west-1":      "ami-0ae5f0dfd13abd9bc", // EU (Ireland).
			"eu-west-2":      "ami-0ad26c54062c16be2", // EU (London).
			"eu-west-3":      "ami-03b0e51270ccad27e", // EU (Paris).
			"me-south-1":     "ami-096ecef5bf7e650c3", // Middle East (Bahrain).
			"sa-east-1":      "ami-0cb8238448799101d", // South America (Sao Paulo).
			"us-east-1":      "ami-065c27623c6d7487f", // US East (N. Virginia).
			"us-east-2":      "ami-07ec0dd544be09fb9", // US East (Ohio).
			"us-west-1":      "ami-07e38410458d6152d", // US West (N. California).
			"us-west-2":      "ami-0387e5911829c8db4", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0ffe2243b98e055d5", // GovCloud (US-East)
			"us-gov-west-1": "ami-0936fa974ff755d79", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.16.13
		Constraint: mustConstraint("1.16"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-09b848d1965bcac64", // Africa (Cape Town).
			"ap-east-1":      "ami-0d8ae48d38aa6b828", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0503df76723c99e5f", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0daffdcd3f9eb32f0", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-0c64ed9c1d97f4894", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-0cebfd00623aac2a3", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-068a13ddd1751e7de", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-0b627d9469ed72ca6", // Canada (Central).
			"eu-central-1":   "ami-063b62b734173cd87", // EU (Frankfurt).
			"eu-north-1":     "ami-0b71dda06df994656", // EU (Stockholm).
			"eu-south-1":     "ami-05c24691f0d47980c", // Europe (Milan).
			"eu-west-1":      "ami-09a7447e3c43872d4", // EU (Ireland).
			"eu-west-2":      "ami-03897338e394beede", // EU (London).
			"eu-west-3":      "ami-0f8e41f304720bb77", // EU (Paris).
			"me-south-1":     "ami-0c79734ffd650f311", // Middle East (Bahrain).
			"sa-east-1":      "ami-07f9783a0257a438f", // South America (Sao Paulo).
			"us-east-1":      "ami-0cea991c2c343d5da", // US East (N. Virginia).
			"us-east-2":      "ami-02e108cd68d254af3", // US East (Ohio).
			"us-west-1":      "ami-03026bd1f5f178895", // US West (N. California).
			"us-west-2":      "ami-0d5a3b3661cf8e969", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-041cb9491fbc60664", // GovCloud (US-East)
			"us-gov-west-1": "ami-021fcf52d9e537f8d", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.17.9
		Constraint: mustConstraint("1.17"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-00fa48e5cb996201e", // Africa (Cape Town).
			"ap-east-1":      "ami-0559b50941175444b", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-01abe1aa23729ddbc", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0a1b3d8800cc10635", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-06230d459643620be", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-0d55bdb0d924129a6", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0538a0821fa5def9a", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-003f7d1160c85495c", // Canada (Central).
			"eu-central-1":   "ami-0d14875522bcaa478", // EU (Frankfurt).
			"eu-north-1":     "ami-09cf4bbfbae9eb09a", // EU (Stockholm).
			"eu-south-1":     "ami-0ba4c1f6187438c65", // Europe (Milan).
			"eu-west-1":      "ami-05f67d6c4a90acb96", // EU (Ireland).
			"eu-west-2":      "ami-02da41dbb05fc1908", // EU (London).
			"eu-west-3":      "ami-01ea22d25b2fb7f75", // EU (Paris).
			"me-south-1":     "ami-0b4ba8b97b3d26d5c", // Middle East (Bahrain).
			"sa-east-1":      "ami-096e3bc15eb88796a", // South America (Sao Paulo).
			"us-east-1":      "ami-02e9f477367b0702e", // US East (N. Virginia).
			"us-east-2":      "ami-0dec6dbc0bc92c320", // US East (Ohio).
			"us-west-1":      "ami-061b43183ceb113f4", // US West (N. California).
			"us-west-2":      "ami-03a8d909816d17ee9", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0be5895c73e55947d", // GovCloud (US-East)
			"us-gov-west-1": "ami-032d4dbd51e007050", // GovCloud (US-West)
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
	KubernetesVersionImageSelector{ // Kubernetes Version 1.14.9
		Constraint: mustConstraint("1.14"),
		ImageSelector: RegionMapImageSelector{
			// AWS partition
			"af-south-1":     "ami-0f1f6e18b1d0b0fa0", // Africa (Cape Town).
			"ap-east-1":      "ami-053d5715a10198024", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0015cdd490150d45c", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-014d4801ecf6563ec", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-078a087230c9eaf2e", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-054d9fa0f0cb68d52", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-07a97b8035b16aceb", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-0b336703e99354fc3", // Canada (Central).
			"eu-central-1":   "ami-04e9f9baf3c59cee5", // EU (Frankfurt).
			"eu-north-1":     "ami-00e746dd95e799449", // EU (Stockholm).
			"eu-south-1":     "ami-08a675a9994c29791", // Europe (Milan).
			"eu-west-1":      "ami-0f8b7e3b9652d2471", // EU (Ireland).
			"eu-west-2":      "ami-0d1ec338b71724a85", // EU (London).
			"eu-west-3":      "ami-0225f225d66f4e376", // EU (Paris).
			"me-south-1":     "ami-08b5df22b6425b512", // Middle East (Bahrain).
			"sa-east-1":      "ami-0fd134f1ba6615c98", // South America (Sao Paulo).
			"us-east-1":      "ami-085a03be6ae060a6f", // US East (N. Virginia).
			"us-east-2":      "ami-0a8d655f910acc426", // US East (Ohio).
			"us-west-1":      "ami-086d2d9bf38807ee9", // US West (N. California).
			"us-west-2":      "ami-04ef04314b56bb963", // US West (Oregon).

			// AWS GovCloud (US) partition
			// Not supported currently.
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.15.11
		Constraint: mustConstraint("1.15"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-0bf5cc3b3c0d320fb", // Africa (Cape Town).
			"ap-east-1":      "ami-02d530cdb99417d90", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-06635cb79824bf67d", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0998f06456baed93d", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-045c67a2c75d36cea", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-0969cc882a8f3d064", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0ed22372b261f5d80", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-07259daf518b979a0", // Canada (Central).
			"eu-central-1":   "ami-06526031f25326b81", // EU (Frankfurt).
			"eu-north-1":     "ami-00fd84e1bdbc053af", // EU (Stockholm).
			"eu-south-1":     "ami-0c7a7de4884cd9cb2", // Europe (Milan).
			"eu-west-1":      "ami-046773f5b634f45e4", // EU (Ireland).
			"eu-west-2":      "ami-05ef9b6eae784065d", // EU (London).
			"eu-west-3":      "ami-0306d8a08375fcb24", // EU (Paris).
			"me-south-1":     "ami-0344d276f5af76029", // Middle East (Bahrain).
			"sa-east-1":      "ami-08369f566915a93d7", // South America (Sao Paulo).
			"us-east-1":      "ami-089035757766d85bc", // US East (N. Virginia).
			"us-east-2":      "ami-0098b27af37cdd5da", // US East (Ohio).
			"us-west-1":      "ami-0b8af535193c8c4ec", // US West (N. California).
			"us-west-2":      "ami-0a7fb37899667e74f", // US West (Oregon).

			// AWS GovCloud (US) partition
			// Not supported currently.
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.16.13
		Constraint: mustConstraint("1.16"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-04eab281805bd8566", // Africa (Cape Town).
			"ap-east-1":      "ami-04ef8ff830854900f", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0d3f241b9c06e0f89", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0ef21b917c84bd57a", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-0b158ffc644d5945d", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-0ce5ea2f61628374a", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0843140b6af9129fd", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-06a1f0d9e45b08386", // Canada (Central).
			"eu-central-1":   "ami-0c32ceb2a82fe184d", // EU (Frankfurt).
			"eu-north-1":     "ami-0b1c906e4a5c53b75", // EU (Stockholm).
			"eu-south-1":     "ami-0c539607135f7fa96", // Europe (Milan).
			"eu-west-1":      "ami-0d97bffe54089059f", // EU (Ireland).
			"eu-west-2":      "ami-04b2d573ebb966ab4", // EU (London).
			"eu-west-3":      "ami-069d58345c946c4a1", // EU (Paris).
			"me-south-1":     "ami-0c1141ea802e54182", // Middle East (Bahrain).
			"sa-east-1":      "ami-0f154941c5f094b0b", // South America (Sao Paulo).
			"us-east-1":      "ami-04d618843f45e581d", // US East (N. Virginia).
			"us-east-2":      "ami-03e6cf37ceedcf15c", // US East (Ohio).
			"us-west-1":      "ami-01e182b8d7a2ca92a", // US West (N. California).
			"us-west-2":      "ami-05f45bbc8a1899c07", // US West (Oregon).

			// AWS GovCloud (US) partition
			// Not supported currently.
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.17.11
		Constraint: mustConstraint("1.17"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-0f473894d74f17483", // Africa (Cape Town).
			"ap-east-1":      "ami-0c02d16656aa7a8ac", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0fed064c3cd4729d1", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0e42eff01f237ffea", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-0e36715290e3fad68", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-0e96b702728842961", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0a463cb4a83eb9ff2", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-00753e913d2d516b2", // Canada (Central).
			"eu-central-1":   "ami-06efc49a5048f9733", // EU (Frankfurt).
			"eu-north-1":     "ami-0c429de40a3a31bc0", // EU (Stockholm).
			"eu-south-1":     "ami-0051cdc83263c30ec", // Europe (Milan).
			"eu-west-1":      "ami-0b657932cc1dacb7d", // EU (Ireland).
			"eu-west-2":      "ami-00d9af01dfb88463c", // EU (London).
			"eu-west-3":      "ami-00f6ad809838dcef3", // EU (Paris).
			"me-south-1":     "ami-0dae7708fd0b0fc59", // Middle East (Bahrain).
			"sa-east-1":      "ami-0ad360fca4a36eb32", // South America (Sao Paulo).
			"us-east-1":      "ami-07ee96b515734922b", // US East (N. Virginia).
			"us-east-2":      "ami-05835ec0d56045247", // US East (Ohio).
			"us-west-1":      "ami-0023a539382ad6461", // US West (N. California).
			"us-west-2":      "ami-0c2396f876327a35a", // US West (Oregon).

			// AWS GovCloud (US) partition
			// Not supported currently.
		},
	},
}

// DefaultARMImages returns an image selector that returns fallback images if no other images are found.
func DefaultARMImages() ImageSelector {
	return defaultARMImages
}
