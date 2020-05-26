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

const (
	DefaultSpotPrice = "0.0" // 0 spot price stands for on-demand instances
)

func constraintForVersion(v string) *semver.Constraints {
	cs, err := semver.NewConstraint(fmt.Sprintf("~%s", v))
	if err != nil {
		emperror.Panic(errors.WrapIff(err, "could not create semver constraint for Kubernetes version %s.x", v))
	}

	return cs
}

// AMIs taken form https://docs.aws.amazon.com/eks/latest/userguide/eks-optimized-ami.html
// nolint: gochecknoglobals
var defaultImageMap = []struct {
	constraint *semver.Constraints
	images     map[string]string
}{
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
			// Kubernetes Version 1.14.9
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
		},
	},
	{
		constraintForVersion("1.15"),
		map[string]string{
			// Kubernetes Version 1.15.10
			"ap-east-1":      "ami-0d591ec9aab8976dc", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-06abd5347585f6519", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-065649f5fee9f227a", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-08805da128ddc2ee1", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-01b5910473e0a2d61", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-0b0bc41a50e8cd33e", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-032ef9dea6ae46809", // Canada (Central).
			"eu-central-1":   "ami-0c9af00bc060dfa76", // EU (Frankfurt).
			"eu-north-1":     "ami-07739287a5dbb16d0", // EU (Stockholm).
			"eu-west-1":      "ami-04bf3ca704bd6b643", // EU (Ireland).
			"eu-west-2":      "ami-0162c7f5400c6ec02", // EU (London).
			"eu-west-3":      "ami-026d2ac4b345304dc", // EU (Paris).
			"me-south-1":     "ami-078805035ccb0040b", // Middle East (Bahrain).
			"sa-east-1":      "ami-0fee705e85dc3ac2c", // South America (Sao Paulo).
			"us-east-1":      "ami-0582e4c984a1e848a", // US East (N. Virginia).
			"us-east-2":      "ami-08880278b3cac5832", // US East (Ohio).
			"us-west-1":      "ami-0b65bc2de276c7db7", // US West (N. California).
			"us-west-2":      "ami-000a48e69e7695a4a", // US West (Oregon).
		},
	},
}

// GetDefaultImageID returns the EKS optimized AMI for given Kubernetes version and region.
func GetDefaultImageID(region string, kubernetesVersion string) (string, error) {
	kubeVersion, err := semver.NewVersion(kubernetesVersion)
	if err != nil {
		return "", errors.WrapIfWithDetails(err, "could not create semver from Kubernetes version", "kubernetesVersion", kubernetesVersion)
	}

	for _, m := range defaultImageMap {
		if m.constraint.Check(kubeVersion) {
			if ami, ok := m.images[region]; ok {
				return ami, nil
			}

			return "", errors.NewWithDetails(
				"no EKS AMI found for Kubernetes version",
				"kubernetesVersion", kubeVersion.String(),
				"region", region,
			)
		}
	}

	return "", errors.Errorf("unsupported Kubernetes version %q", kubeVersion)
}
