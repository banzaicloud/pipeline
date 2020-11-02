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
			"af-south-1":     "ami-016b65c518c4d3419", // Africa (Cape Town).
			"ap-east-1":      "ami-03e902f04ba30efb5", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0c04b46d5b7c69191", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-08449bd0854b50a4a", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-0cb794a6e40561558", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-087315adc4086bcef", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0c16646c85f24652d", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-048c87a0c93e3bd79", // Canada (Central).
			"eu-central-1":   "ami-0e941b21b7ccd9fff", // EU (Frankfurt).
			"eu-north-1":     "ami-059855484758944d6", // EU (Stockholm).
			"eu-south-1":     "ami-08f6162a9cfbb1e0c", // Europe (Milan).
			"eu-west-1":      "ami-0f5a80cfb11fbda5b", // EU (Ireland).
			"eu-west-2":      "ami-02832b3dffa1f8432", // EU (London).
			"eu-west-3":      "ami-0cac45c390722111a", // EU (Paris).
			"me-south-1":     "ami-0adacffc1cb1840cd", // Middle East (Bahrain).
			"sa-east-1":      "ami-038f4c5db2d62203f", // South America (Sao Paulo).
			"us-east-1":      "ami-0fc7c80d4d288ab2c", // US East (N. Virginia).
			"us-east-2":      "ami-08984d8491de17ca0", // US East (Ohio).
			"us-west-1":      "ami-0e796ccc93168da3a", // US West (N. California).
			"us-west-2":      "ami-0f8fb024d10446ce1", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0ad32e493bd4f3d40", // GovCloud (US-East)
			"us-gov-west-1": "ami-0d34dbb85e5c6280d", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.15.11
		Constraint: mustConstraint("1.15"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-04072d5b282afb06b", // Africa (Cape Town).
			"ap-east-1":      "ami-05581693aaba1f770", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0fad58b4886213487", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-03de2b93e19fc1e32", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-0aaecfe8a45d922dd", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-086f3dbe335a7bc71", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0d93857f3c351a307", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-04957e41a52bfc38f", // Canada (Central).
			"eu-central-1":   "ami-010266493fc743a3d", // EU (Frankfurt).
			"eu-north-1":     "ami-0b88d4ed6217a3018", // EU (Stockholm).
			"eu-south-1":     "ami-0e1ac6e3bac59db7a", // Europe (Milan).
			"eu-west-1":      "ami-0e454bb2832574b25", // EU (Ireland).
			"eu-west-2":      "ami-0af730da10ac8b0b7", // EU (London).
			"eu-west-3":      "ami-01c1194e8e2743a0b", // EU (Paris).
			"me-south-1":     "ami-07ba1047a42f3ff55", // Middle East (Bahrain).
			"sa-east-1":      "ami-0fae027dad99aaefa", // South America (Sao Paulo).
			"us-east-1":      "ami-0ef76ba092ce4e253", // US East (N. Virginia).
			"us-east-2":      "ami-0eda59dcbd424485d", // US East (Ohio).
			"us-west-1":      "ami-0a8d9b717b89fdd17", // US West (N. California).
			"us-west-2":      "ami-0d552892d1327b499", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-019e9af034a3de556", // GovCloud (US-East)
			"us-gov-west-1": "ami-05a39266f734948da", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.16.13
		Constraint: mustConstraint("1.16"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-004caadd8fd18aee6", // Africa (Cape Town).
			"ap-east-1":      "ami-0bf06ed6ab00bdc8b", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0e75cf37211a7b9c1", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-024c291fb23ec5253", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-071bba9de77f5666a", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-02fdbc9449e2edac5", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0c37aec140141b5fa", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-0215927206ce98fed", // Canada (Central).
			"eu-central-1":   "ami-0951ecbcce367c97b", // EU (Frankfurt).
			"eu-north-1":     "ami-0b483af7cd3079758", // EU (Stockholm).
			"eu-south-1":     "ami-0cfe17838ad3a6710", // Europe (Milan).
			"eu-west-1":      "ami-0313d49570831d7f4", // EU (Ireland).
			"eu-west-2":      "ami-0afeda8fcb3e79571", // EU (London).
			"eu-west-3":      "ami-0d1badbc7843bf2eb", // EU (Paris).
			"me-south-1":     "ami-06c259406da782e50", // Middle East (Bahrain).
			"sa-east-1":      "ami-073549ba2423737e4", // South America (Sao Paulo).
			"us-east-1":      "ami-0b0dffffb5ba92f97", // US East (N. Virginia).
			"us-east-2":      "ami-0f8d6052f6e3a19d2", // US East (Ohio).
			"us-west-1":      "ami-009a9a6ad7a1670c6", // US West (N. California).
			"us-west-2":      "ami-04445f1c3b2132901", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0ea73d90aca4dd1e4", // GovCloud (US-East)
			"us-gov-west-1": "ami-0d3eab4e3b29418bf", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.17.11
		Constraint: mustConstraint("1.17"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-00e923e1e72dd6127", // Africa (Cape Town).
			"ap-east-1":      "ami-068cd0572b507ae91", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0aa15614ef924fd1e", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-025592e84db381916", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-084ea7596600d9844", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-056a5f106add4d37a", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0cd8562db082e8c1a", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-032321703c4a51e39", // Canada (Central).
			"eu-central-1":   "ami-06cfd5b2a2d58e09a", // EU (Frankfurt).
			"eu-north-1":     "ami-067bfa3d76de7a6b7", // EU (Stockholm).
			"eu-south-1":     "ami-03c6f34023cdadbfe", // Europe (Milan).
			"eu-west-1":      "ami-0c504dda1302b182f", // EU (Ireland).
			"eu-west-2":      "ami-08c210420094a901b", // EU (London).
			"eu-west-3":      "ami-0c1f4efd77302f897", // EU (Paris).
			"me-south-1":     "ami-01b2fefee93216e1a", // Middle East (Bahrain).
			"sa-east-1":      "ami-040a741cd6e254cad", // South America (Sao Paulo).
			"us-east-1":      "ami-07250434f8a7bc5f1", // US East (N. Virginia).
			"us-east-2":      "ami-0135903686f192ffe", // US East (Ohio).
			"us-west-1":      "ami-05bfd72ad17ebedb8", // US West (N. California).
			"us-west-2":      "ami-0c62450bce8f4f57f", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-07fedeaa820972f24", // GovCloud (US-East)
			"us-gov-west-1": "ami-0efd27cb74621d6b0", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.18.8
		Constraint: mustConstraint("1.18"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-0dcfa2d757494da7c", // Africa (Cape Town).
			"ap-east-1":      "ami-0824aac4c54c763d3", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0e9f5606a6d10ffb1", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-09b14b49f6e5be4a1", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-0042bc79e92fb3c8a", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-0dadf836fc8220165", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0b53169adb5906e18", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-0becc01e0dd0dd238", // Canada (Central).
			"eu-central-1":   "ami-045e4ecd708ac12ba", // EU (Frankfurt).
			"eu-north-1":     "ami-0a674c329567c6456", // EU (Stockholm).
			"eu-south-1":     "ami-0d236a46607b78f5e", // Europe (Milan).
			"eu-west-1":      "ami-0ca9e57915fd7e017", // EU (Ireland).
			"eu-west-2":      "ami-062c2b6eee26e5603", // EU (London).
			"eu-west-3":      "ami-02444825d174fbd7b", // EU (Paris).
			"me-south-1":     "ami-058f6a482ed37d011", // Middle East (Bahrain).
			"sa-east-1":      "ami-0bfae48e8718fde5f", // South America (Sao Paulo).
			"us-east-1":      "ami-0fae38e27c6113140", // US East (N. Virginia).
			"us-east-2":      "ami-0dc6bc43da1b962d8", // US East (Ohio).
			"us-west-1":      "ami-002e04ca6d86d255e", // US West (N. California).
			"us-west-2":      "ami-04f0f3d381d07e0b6", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0880473286868dd3c", // GovCloud (US-East)
			"us-gov-west-1": "ami-0b865a9d86ba6b983", // GovCloud (US-West)
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
			"af-south-1":     "ami-0eecf979f04ca76e5", // Africa (Cape Town).
			"ap-east-1":      "ami-09d7fd006cd9dba15", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-084a66776b45850ab", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-09de2d86527e4570a", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-0210cdb1b32ec9347", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-045c170258419ae8d", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0e3d484acade46986", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-0799b97b5c40faaa2", // Canada (Central).
			"eu-central-1":   "ami-06e2463cc299c278f", // EU (Frankfurt).
			"eu-north-1":     "ami-055fa894900c0acd3", // EU (Stockholm).
			"eu-south-1":     "ami-008e642b7e268220a", // Europe (Milan).
			"eu-west-1":      "ami-0ce46866af7b26c7f", // EU (Ireland).
			"eu-west-2":      "ami-05dd921cb6514d060", // EU (London).
			"eu-west-3":      "ami-045d69d2572bc0de5", // EU (Paris).
			"me-south-1":     "ami-074983eaf7057e369", // Middle East (Bahrain).
			"sa-east-1":      "ami-06f5a0c6b643f4996", // South America (Sao Paulo).
			"us-east-1":      "ami-0d31dd0c5a95398b1", // US East (N. Virginia).
			"us-east-2":      "ami-0722de2279d93a431", // US East (Ohio).
			"us-west-1":      "ami-051e137727a70bc73", // US West (N. California).
			"us-west-2":      "ami-0eabb2e84f3c9b9ca", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0db5a00b3c9bbade2", // GovCloud (US-East)
			"us-gov-west-1": "ami-053ff8715e668d894", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.15.11
		Constraint: mustConstraint("1.15"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-0fbf3b7d6e25ebcb7", // Africa (Cape Town).
			"ap-east-1":      "ami-0258ddf6cd6b889a7", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-058604eb4bd017c81", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0b46871c72bb1c164", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-0c7ce75dc7cb67a46", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-08c283d2e28651e4c", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0e2e933dd75ee43b6", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-002e10f0559d63a23", // Canada (Central).
			"eu-central-1":   "ami-0b2e45a5ff7fc8ad9", // EU (Frankfurt).
			"eu-north-1":     "ami-071c1f28c80bb1a28", // EU (Stockholm).
			"eu-south-1":     "ami-0123fb6eb61e82c8d", // Europe (Milan).
			"eu-west-1":      "ami-0530d6e90e791e4ba", // EU (Ireland).
			"eu-west-2":      "ami-044b8f3cf798f7a3f", // EU (London).
			"eu-west-3":      "ami-0e348ca5974e00348", // EU (Paris).
			"me-south-1":     "ami-0d4d12ad532bb0d02", // Middle East (Bahrain).
			"sa-east-1":      "ami-049ca6966937793f7", // South America (Sao Paulo).
			"us-east-1":      "ami-081bcc9e174dde4c0", // US East (N. Virginia).
			"us-east-2":      "ami-006b8dd0737d21182", // US East (Ohio).
			"us-west-1":      "ami-03bba6885815a8cfe", // US West (N. California).
			"us-west-2":      "ami-01169f46f1fe4f9a3", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0944fb68223162264", // GovCloud (US-East)
			"us-gov-west-1": "ami-0c9e2aa7d6db2d70d", // GovCloud (US-West)

		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.16.13
		Constraint: mustConstraint("1.16"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-0534f90479cfb897f", // Africa (Cape Town).
			"ap-east-1":      "ami-0ee7e429b4d4165c7", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-080e5822bd4b701eb", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-081e15cf1a67d61d9", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-038a264fdc76a0a09", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-09581c7d7938144f3", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-042cc228a69cd254d", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-00949e04558284a26", // Canada (Central).
			"eu-central-1":   "ami-07bd8ae99a9171cb6", // EU (Frankfurt).
			"eu-north-1":     "ami-0c1b576244c1ec555", // EU (Stockholm).
			"eu-south-1":     "ami-06eca0cb4ceb69364", // Europe (Milan).
			"eu-west-1":      "ami-0afe11aa8f6fac118", // EU (Ireland).
			"eu-west-2":      "ami-0f2f219bcd813f912", // EU (London).
			"eu-west-3":      "ami-0d3e410e181fab211", // EU (Paris).
			"me-south-1":     "ami-0ecd0fd38025d712c", // Middle East (Bahrain).
			"sa-east-1":      "ami-0daa3056401e8a590", // South America (Sao Paulo)
			"us-east-1":      "ami-06b186d1af79c4b6e", // US East (N. Virginia).
			"us-east-2":      "ami-0a8c727d07d6344e0", // US East (Ohio).
			"us-west-1":      "ami-059870ea217f996c1", // US West (N. California).
			"us-west-2":      "ami-0e75129a886e69282", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0b4c610b5327cf9ad", // GovCloud (US-East)
			"us-gov-west-1": "ami-01e21ce0467d4d4a2", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.17.11
		Constraint: mustConstraint("1.17"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-01c0ea52c2b407bd4", // Africa (Cape Town).
			"ap-east-1":      "ami-05aee69bec8d1f3f6", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-085e83d0565b79dd9", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-07b8435972ba1770b", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-06267c581bdb74056", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-0d4bf18587d78de6b", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0f3e6cade3e5cefc9", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-0b931425637cbd377", // Canada (Central).
			"eu-central-1":   "ami-0e338e38b20ccc545", // EU (Frankfurt).
			"eu-north-1":     "ami-02a6f40845860d59a", // EU (Stockholm).
			"eu-south-1":     "ami-0eaef85df91031259", // Europe (Milan).
			"eu-west-1":      "ami-0f48f36867f68d787", // EU (Ireland).
			"eu-west-2":      "ami-0b523549f45978aac", // EU (London).
			"eu-west-3":      "ami-07e8992e3cad5d66e", // EU (Paris).
			"me-south-1":     "ami-09e229ec0dc3f0e00", // Middle East (Bahrain).
			"sa-east-1":      "ami-064999c4081018e7f", // South America (Sao Paulo)
			"us-east-1":      "ami-00cff21748681dd2b", // US East (N. Virginia).
			"us-east-2":      "ami-0cccbc858cb8b7792", // US East (Ohio).
			"us-west-1":      "ami-08da01741f6ea2d10", // US West (N. California).
			"us-west-2":      "ami-08655818c336c3a28", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0805d2ba6e4ce10bb", // GovCloud (US-East)
			"us-gov-west-1": "ami-0329485cb9be5e531", // GovCloud (US-West)
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.18.8
		Constraint: mustConstraint("1.18"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-02d60c3325c02dae2", // Africa (Cape Town).
			"ap-east-1":      "ami-0088283db01d5b21f", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0f4710241473eddd0", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0efc991c6fe02895a", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-095ff396b89e735ce", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-00d8e812549c9c3d8", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0b51b23678b6d25cd", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-0d42560ff65e2abf1", // Canada (Central).
			"eu-central-1":   "ami-0ad2a3ab0f2e50c18", // EU (Frankfurt).
			"eu-north-1":     "ami-072d2239da787b8c3", // EU (Stockholm).
			"eu-south-1":     "ami-0994afd26ff756ef3", // Europe (Milan).
			"eu-west-1":      "ami-0969f51a73874a795", // EU (Ireland).
			"eu-west-2":      "ami-09fad23b80bb14ee2", // EU (London).
			"eu-west-3":      "ami-045b9b022a23added", // EU (Paris).
			"me-south-1":     "ami-0e38acd0225fe7015", // Middle East (Bahrain).
			"sa-east-1":      "ami-0d831f9c337257a39", // South America (Sao Paulo).
			"us-east-1":      "ami-07976b5eda30dedb8", // US East (N. Virginia).
			"us-east-2":      "ami-0b3cc8ed77576ba67", // US East (Ohio).
			"us-west-1":      "ami-0dc99e95df35751f3", // US West (N. California).
			"us-west-2":      "ami-0f4175b08dd293e71", // US West (Oregon).

			// AWS GovCloud (US) partition
			"us-gov-east-1": "ami-0a8a170d92f2eddf5", // GovCloud (US-East)
			"us-gov-west-1": "ami-014b5ea8259414c88", // GovCloud (US-West)
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
	KubernetesVersionImageSelector{ // Kubernetes Version 1.18.8
		Constraint: mustConstraint("1.18"),
		ImageSelector: RegionMapImageSelector{
			"af-south-1":     "ami-0efea21cfce54b4cc", // Africa (Cape Town).
			"ap-east-1":      "ami-01c60b3e8ad6c452a", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-05b66c52d9ade6141", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-009a260a44035e32d", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-06f3a1ca6c3647d06", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-0db4a08f12004e7d6", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-08e15a20d904474d5", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-0c74928ca2e423e9a", // Canada (Central).
			"eu-central-1":   "ami-0ed8f6e4ae493b982", // EU (Frankfurt).
			"eu-north-1":     "ami-0c2900a5ea57b3978", // EU (Stockholm).
			"eu-south-1":     "ami-010275e585a2d21e3", // Europe (Milan).
			"eu-west-1":      "ami-07b445f631b2c5d80", // EU (Ireland).
			"eu-west-2":      "ami-0366df046883d6efc", // EU (London).
			"eu-west-3":      "ami-090942489187eadf5", // EU (Paris).
			"me-south-1":     "ami-01d54c83192263f71", // Middle East (Bahrain).
			"sa-east-1":      "ami-060eb314877301951", // South America (Sao Paulo).
			"us-east-1":      "ami-0e3b0211ae52b6cf5", // US East (N. Virginia).
			"us-east-2":      "ami-0edf7fbe0d28d5488", // US East (Ohio).
			"us-west-1":      "ami-05e3edc366c5db691", // US West (N. California).
			"us-west-2":      "ami-07d922e2104916167", // US West (Oregon).

			// AWS GovCloud (US) partition
			// Not supported currently.
		},
	},
}

// DefaultARMImages returns an image selector that returns fallback images if no other images are found.
func DefaultARMImages() ImageSelector {
	return defaultARMImages
}
