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

import "github.com/aws/aws-sdk-go/aws/endpoints"

// ### [ Constants to EKS cluster default values ] ### //
const (
	DefaultInstanceType = "m4.xlarge"
	DefaultSpotPrice    = "0.0" // 0 spot price stands for on-demand instances
	DefaultRegion       = endpoints.UsWest2RegionID
	DefaultK8sVersion   = "1.11"
)

// DefaultImages in each supported location in EC2 (from https://docs.aws.amazon.com/eks/latest/userguide/launch-workers.html)
var DefaultImages = map[string]map[string]string{
	"1.10": {
		endpoints.UsEast1RegionID:      "ami-04358410d28eaab63",
		endpoints.UsEast2RegionID:      "ami-0b779e8ab57655b4b",
		endpoints.UsWest2RegionID:      "ami-09e1df3bad220af0b",
		endpoints.EuWest1RegionID:      "ami-0de10c614955da932",
		endpoints.EuNorth1RegionID:     "ami-068b8a1efffd30eda",
		endpoints.EuCentral1RegionID:   "ami-08eb700778f03ea94",
		endpoints.ApNortheast1RegionID: "ami-06398bdd37d76571d",
		endpoints.ApNortheast2RegionID: "ami-08a87e0a7c32fa649",
		endpoints.ApSoutheast1RegionID: "ami-0ac3510e44b5bf8ef",
		endpoints.ApSoutheast2RegionID: "ami-0d2c929ace88cfebe",
	},
	"1.11": {
		endpoints.UsEast1RegionID:      "ami-0c24db5df6badc35a",
		endpoints.UsEast2RegionID:      "ami-0c2e8d28b1f854c68",
		endpoints.UsWest2RegionID:      "ami-0a2abab4107669c1b",
		endpoints.EuWest1RegionID:      "ami-01e08d22b9439c15a",
		endpoints.EuNorth1RegionID:     "ami-06ee67302ab7cf838",
		endpoints.EuCentral1RegionID:   "ami-010caa98bae9a09e2",
		endpoints.ApNortheast1RegionID: "ami-0f0e8066383e7a2cb",
		endpoints.ApNortheast2RegionID: "ami-0b7baa90de70f683f",
		endpoints.ApSoutheast1RegionID: "ami-019966ed970c18502",
		endpoints.ApSoutheast2RegionID: "ami-06ade0abbd8eca425",
	},
}
