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
)

// ### [ Constants to EKS cluster default values ] ### //
const (
	DefaultSpotPrice = "0.0" // 0 spot price stands for on-demand instances
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
			"ap-east-1":      "ami-0e18a9300e3385073", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0c5f7ed15d3ad01bf", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0f8af14150481eea9", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-0c3982d02380f4057", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-0fb4518280a63a4f9", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0cfcbd8cd1dcf0b86", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-05ffff48d99f02a77", // Canada (Central).
			"eu-central-1":   "ami-062b278aa4f541381", // EU (Frankfurt).
			"eu-north-1":     "ami-0ba8e7f06e6a019d2", // EU (Stockholm).
			"eu-west-1":      "ami-045c779913c08c97f", // EU (Ireland).
			"eu-west-2":      "ami-0c4ac9257aa949170", // EU (London).
			"eu-west-3":      "ami-0ab23a43c29006f2c", // EU (Paris).
			"me-south-1":     "ami-0f82cb40b1fffed23", // Middle East (Bahrain).
			"sa-east-1":      "ami-0d3c5fab22805c5f3", // South America (Sao Paulo).
			"us-east-1":      "ami-0010e840204e3182b", // US East (N. Virginia).
			"us-east-2":      "ami-028dd95165fbe9bc1", // US East (Ohio).
			"us-west-1":      "ami-08dc171b543edad80", // US West (N. California).
			"us-west-2":      "ami-0731c2cf494afcd99", // US West (Oregon).
		},
	},
	{
		constraintForVersion("1.12"),
		map[string]string{
			// Kubernetes Version 1.12.10
			"ap-east-1":      "ami-0d8112a829ab1dbe3", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-02355b50897aa3aa1", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-02c654f7a3055a0b5", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-03a8ac8740d6ddcd1", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-0bf3d8667e090925b", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0229b20de901562bb", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-072d30b45d3515f6a", // Canada (Central).
			"eu-central-1":   "ami-033d1d6ff9131d9f1", // EU (Frankfurt).
			"eu-north-1":     "ami-04e160b95d368ebc9", // EU (Stockholm).
			"eu-west-1":      "ami-04b0486b375074f76", // EU (Ireland).
			"eu-west-2":      "ami-0aed33d48fb4e1f41", // EU (London).
			"eu-west-3":      "ami-048395c260672dd76", // EU (Paris).
			"me-south-1":     "ami-05c110c75b8ae1fe3", // Middle East (Bahrain).
			"sa-east-1":      "ami-02fcd63b41f21e854", // South America (Sao Paulo).
			"us-east-1":      "ami-0ad6567ae0a817577", // US East (N. Virginia).
			"us-east-2":      "ami-0243b86bf17bdf282", // US East (Ohio).
			"us-west-1":      "ami-0bf8e913be03a3819", // US West (N. California).
			"us-west-2":      "ami-0ec0eebfe09f8a641", // US West (Oregon).
		},
	},
	{
		constraintForVersion("1.13"),
		map[string]string{
			// Kubernetes Version 1.13.11
			"ap-east-1":      "ami-061c919d6ecc3fdb4", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-01fd7f32ab8a9e032", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-053959c7a4a9cb654", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-0baa81231c278c1ac", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-091a252b3e9cabcc2", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0b667ccbbae9214e3", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-0808a5ff743eb2806", // Canada (Central).
			"eu-central-1":   "ami-0a9ad7a4ae50e8e77", // EU (Frankfurt).
			"eu-north-1":     "ami-0b9403c917e4f92b5", // EU (Stockholm).
			"eu-west-1":      "ami-08684dce117829aa8", // EU (Ireland).
			"eu-west-2":      "ami-07bf4afe6ca486eeb", // EU (London).
			"eu-west-3":      "ami-095de5b6bd8b1acf0", // EU (Paris).
			"me-south-1":     "ami-02ec1b153ae90c2c3", // Middle East (Bahrain).
			"sa-east-1":      "ami-035e63ad35c591df8", // South America (Sao Paulo).
			"us-east-1":      "ami-0795ae6584e7f8070", // US East (N. Virginia).
			"us-east-2":      "ami-01505c630227fa3f8", // US East (Ohio).
			"us-west-1":      "ami-02f3579ca4683a2ed", // US West (N. California).
			"us-west-2":      "ami-04e247c4613de71fa", // US West (Oregon).
		},
	},
	{
		constraintForVersion("1.14"),
		map[string]string{
			// Kubernetes Version 1.14.7
			"ap-east-1":      "ami-0af3a70f827304d17", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0b60cbd90564dfe00", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0cf70ba01dfd0f782", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-0d275f57a60281ccc", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-0159ec8365aea1724", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-07e2a96e251e970bd", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-0ef56ecc6435d1f65", // Canada (Central).
			"eu-central-1":   "ami-03fbd442f4f3aa689", // EU (Frankfurt).
			"eu-north-1":     "ami-01feb408eb7fc7e23", // EU (Stockholm).
			"eu-west-1":      "ami-02dca57ad67c7bf57", // EU (Ireland).
			"eu-west-2":      "ami-0a69fbeff04e330e9", // EU (London).
			"eu-west-3":      "ami-074b0da576fa9f5c9", // EU (Paris).
			"me-south-1":     "ami-0fc6f1ff5cd458c95", // Middle East (Bahrain).
			"sa-east-1":      "ami-010ffc66e06c843b2", // South America (Sao Paulo).
			"us-east-1":      "ami-07d6c8e62ce328a10", // US East (N. Virginia).
			"us-east-2":      "ami-053250833d1030033", // US East (Ohio).
			"us-west-1":      "ami-062d2cddf8747e025", // US West (N. California).
			"us-west-2":      "ami-07be7092831897fd6", // US West (Oregon).
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

			return "", errors.Errorf("no EKS AMI found for Kubernetes version %q in region %q", kubeVersion, region)
		}
	}

	return "", fmt.Errorf("unsupported Kubernetes version %q", kubeVersion)
}
