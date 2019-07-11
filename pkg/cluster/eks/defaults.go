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

	"github.com/Masterminds/semver"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/goph/emperror"
)

// ### [ Constants to EKS cluster default values ] ### //
const (
	DefaultInstanceType = "m4.xlarge"
	DefaultSpotPrice    = "0.0" // 0 spot price stands for on-demand instances
	DefaultRegion       = endpoints.UsWest2RegionID
	DefaultK8sVersion   = "1.13.7"
)

var (
	// nolint: gochecknoglobals
	amis110 = map[string]string{
		endpoints.UsEast2RegionID:      "ami-0295a10750423107d",
		endpoints.UsEast1RegionID:      "ami-05c9fba3332ccbc43",
		endpoints.UsWest2RegionID:      "ami-0fc349241eb7b1222",
		endpoints.ApSouth1RegionID:     "ami-0a183946b284a9841",
		endpoints.ApNortheast1RegionID: "ami-0f93f5579e6e79e96",
		endpoints.ApNortheast2RegionID: "ami-0412ddfd70b9c54bd",
		endpoints.ApSoutheast1RegionID: "ami-0538e8e564078659c",
		endpoints.ApSoutheast2RegionID: "ami-009caed75bdc3a2f0",
		endpoints.EuCentral1RegionID:   "ami-032fc49751b7a5f83",
		endpoints.EuWest1RegionID:      "ami-03f9c85cd73fb9f4a",
		endpoints.EuWest2RegionID:      "ami-05c9cec73d17bf97f",
		endpoints.EuWest3RegionID:      "ami-0df95e4cd302d42f7",
		endpoints.EuNorth1RegionID:     "ami-0ef218c64404e4bdf",
	}

	// nolint: gochecknoglobals
	amis111 = map[string]string{
		endpoints.UsEast2RegionID:      "ami-088dad958fbfa643e",
		endpoints.UsEast1RegionID:      "ami-053e2ac42d872cc20",
		endpoints.UsWest2RegionID:      "ami-0743039b7c66a18f5",
		endpoints.ApSouth1RegionID:     "ami-01d152acba5840ba2",
		endpoints.ApNortheast1RegionID: "ami-07765e1384d2e372c",
		endpoints.ApNortheast2RegionID: "ami-0656df091f27461cd",
		endpoints.ApSoutheast1RegionID: "ami-084e9f3625a1a4a09",
		endpoints.ApSoutheast2RegionID: "ami-03050c93b7e745696",
		endpoints.EuCentral1RegionID:   "ami-020f08a17c3c4251c",
		endpoints.EuWest1RegionID:      "ami-07d0c92a42077ec9b",
		endpoints.EuWest2RegionID:      "ami-0ff8a4dc1632ee425",
		endpoints.EuWest3RegionID:      "ami-0569332dde21e3f1a",
		endpoints.EuNorth1RegionID:     "ami-0fc8c638bc80fcecf",
	}

	// nolint: gochecknoglobals
	amis112 = map[string]string{
		endpoints.UsEast2RegionID:      "ami-0e8d353285e26a68c",
		endpoints.UsEast1RegionID:      "ami-0200e65a38edfb7e1",
		endpoints.UsWest2RegionID:      "ami-0f11fd98b02f12a4c",
		endpoints.ApSouth1RegionID:     "ami-0644de45344ce867e",
		endpoints.ApNortheast1RegionID: "ami-0dfbca8d183884f02",
		endpoints.ApNortheast2RegionID: "ami-0a9d12fe9c2a31876",
		endpoints.ApSoutheast1RegionID: "ami-040bdde117f3828ab",
		endpoints.ApSoutheast2RegionID: "ami-01bfe815f644becc0",
		endpoints.EuCentral1RegionID:   "ami-09ed3f40a2b3c11f1",
		endpoints.EuWest1RegionID:      "ami-091fc251b67b776c3",
		endpoints.EuWest2RegionID:      "ami-0bc8d0262346bd65e",
		endpoints.EuWest3RegionID:      "ami-0084dea61e480763e",
		endpoints.EuNorth1RegionID:     "ami-022cd6a50742d611a",
	}

	// nolint: gochecknoglobals
	amis113 = map[string]string{
		endpoints.UsEast2RegionID:      "ami-07ebcae043cf995aa",
		endpoints.UsEast1RegionID:      "ami-08c4955bcc43b124e",
		endpoints.UsWest2RegionID:      "ami-089d3b6350c1769a6",
		endpoints.ApSouth1RegionID:     "ami-0410a80d323371237",
		endpoints.ApNortheast1RegionID: "ami-04c0f02f5e148c80a",
		endpoints.ApNortheast2RegionID: "ami-0b7997a20f8424fb1",
		endpoints.ApSoutheast1RegionID: "ami-087e0fca60fb5737a",
		endpoints.ApSoutheast2RegionID: "ami-082dfea752d9163f6",
		endpoints.EuCentral1RegionID:   "ami-02d5e7ca7bc498ef9",
		endpoints.EuWest1RegionID:      "ami-09bbefc07310f7914",
		endpoints.EuWest2RegionID:      "ami-0f03516f22468f14e",
		endpoints.EuWest3RegionID:      "ami-051015c2c2b73aaea",
		endpoints.EuNorth1RegionID:     "ami-0c31ee32297e7397d",
	}
)

// GetDefaultImageID returns the EKS optimized AMI for given Kubernetes version and region
func GetDefaultImageID(region, kubernetesVersion string) (string, error) {
	// AMIs taken form https://docs.aws.amazon.com/eks/latest/userguide/eks-optimized-ami.html
	constraint110, err := semver.NewConstraint("~1.10")
	if err != nil {
		return "", emperror.Wrap(err, "could not create semver constraint for Kubernetes version 1.10.x")
	}

	constraint111, err := semver.NewConstraint("~1.11")
	if err != nil {
		return "", emperror.Wrap(err, "could not create semver constraint for Kubernetes version 1.11.x")
	}

	constraint112, err := semver.NewConstraint("~1.12")
	if err != nil {
		return "", emperror.Wrap(err, "could not create semver constraint for Kubernetes version 1.12.x")
	}

	constraint113, err := semver.NewConstraint("~1.13")
	if err != nil {
		return "", emperror.Wrap(err, "could not create semver constraint for Kubernetes version 1.13.x")
	}

	kubeVersion, err := semver.NewVersion(kubernetesVersion)
	if err != nil {
		return "", emperror.WrapWith(err, "could not create semver from Kubernetes version", "kubernetesVersion", kubernetesVersion)
	}

	var ami string
	var ok bool

	switch {
	case constraint110.Check(kubeVersion):
		ami, ok = amis110[region]
	case constraint111.Check(kubeVersion):
		ami, ok = amis111[region]
	case constraint112.Check(kubeVersion):
		ami, ok = amis112[region]

	case constraint113.Check(kubeVersion):
		ami, ok = amis113[region]
	default:
		return "", fmt.Errorf("unsupported Kubernetes version %q", kubeVersion)
	}

	if !ok {
		return "", fmt.Errorf("no EKS AMI found for Kubernetes version %q in region %q", kubeVersion, region)

	}

	return ami, nil
}
