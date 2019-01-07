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
)

// DefaultImages in each supported location in EC2 (from https://docs.aws.amazon.com/eks/latest/userguide/launch-workers.html)
var DefaultImages = map[string]string{
	endpoints.UsEast1RegionID:      "ami-027792c3cc6de7b5b",
	endpoints.UsEast2RegionID:      "ami-036130f4127a367f7",
	endpoints.UsWest2RegionID:      "ami-07af9511082779ae7",
	endpoints.EuWest1RegionID:      "ami-03612357ac9da2c7d",
	endpoints.EuNorth1RegionID:     "ami-04b0f84e5a05e0b30",
	endpoints.EuCentral1RegionID:   "ami-06d069282a5fea248",
	endpoints.ApNortheast1RegionID: "ami-06f4af3742fca5998",
	endpoints.ApSoutheast1RegionID: "ami-0bc97856f0dd86d41",
	endpoints.ApSoutheast2RegionID: "ami-05d25b3f16e685c2e",
}
