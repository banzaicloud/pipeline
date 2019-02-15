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
// nolint: gochecknoglobals
var DefaultImages = map[string]map[string]string{
	"1.10": {
		endpoints.UsEast1RegionID:      "ami-0de0b13514617a168",
		endpoints.UsEast2RegionID:      "ami-0d885462fa1a40e3a",
		endpoints.UsWest2RegionID:      "ami-0e36fae01a5fa0d76",
		endpoints.EuWest1RegionID:      "ami-076c1952dd7a28909",
		endpoints.EuWest2RegionID:      "ami-0bfa0f971add9fb2f",
		endpoints.EuWest3RegionID:      "ami-0f0e4bda9786ec624",
		endpoints.EuNorth1RegionID:     "ami-0e1d5399bfbe402e0",
		endpoints.EuCentral1RegionID:   "ami-074583f8d5a05e27b",
		endpoints.ApNortheast1RegionID: "ami-049090cdbc5e3c080",
		endpoints.ApNortheast2RegionID: "ami-0b39dee42365df927",
		endpoints.ApSoutheast1RegionID: "ami-0a3df91af7c8225db",
		endpoints.ApSoutheast2RegionID: "ami-0f4d387d27ad36792",
		endpoints.ApSouth1RegionID:     "ami-0c2a98be00f0b5bb4",
	},
	"1.11": {
		endpoints.UsEast1RegionID:      "ami-0c5b63ec54dd3fc38",
		endpoints.UsEast2RegionID:      "ami-0b10ebfc82e446296",
		endpoints.UsWest2RegionID:      "ami-081099ec932b99961",
		endpoints.EuWest1RegionID:      "ami-0b469c0fef0445d29",
		endpoints.EuWest2RegionID:      "ami-0420d737e57af699c",
		endpoints.EuWest3RegionID:      "ami-0f5a996749bdfa436",
		endpoints.EuNorth1RegionID:     "ami-0da59d86953d1c266",
		endpoints.EuCentral1RegionID:   "ami-05e062a123092066a",
		endpoints.ApNortheast1RegionID: "ami-04ef881404deec134",
		endpoints.ApNortheast2RegionID: "ami-0d87105164496b94b",
		endpoints.ApSoutheast1RegionID: "ami-030c789a75c8bfbca",
		endpoints.ApSoutheast2RegionID: "ami-0a9b90002a9a1c111",
		endpoints.ApSouth1RegionID:     "ami-033ea52f19ce48998",
	},
}
