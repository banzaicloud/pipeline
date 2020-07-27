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

package pkeaws

// Default PKE image: Ubuntu 18.04
// These are plain images, so PKE will be installed during the node bootstrapping.
// nolint: gochecknoglobals, deadcode
var defaultImages = RegionMapImageSelector{
	// AWS partition
	"ap-east-1":      "ami-c790d6b6",          // Asia Pacific (Hong Kong).
	"ap-northeast-1": "ami-0278fe6949f6b1a06", // Asia Pacific (Tokyo).
	"ap-northeast-2": "ami-00edfb46b107f643c", // Asia Pacific (Seoul).
	"ap-southeast-1": "ami-0f7719e8b7ba25c61", // Asia Pacific (Singapore).
	"ap-southeast-2": "ami-04fcc97b5f6edcd89", // Asia Pacific (Sydney).
	"ap-south-1":     "ami-0b44050b2d893d5f7", // Asia Pacific (Mumbai).
	"ca-central-1":   "ami-0edd51cc29813e254", // Canada (Central).
	"eu-central-1":   "ami-0e342d72b12109f91", // EU (Frankfurt).
	"eu-north-1":     "ami-050981837962d44ac", // EU (Stockholm).
	"eu-west-1":      "ami-0701e7be9b2a77600", // EU (Ireland).
	"eu-west-2":      "ami-0eb89db7593b5d434", // EU (London).
	"eu-west-3":      "ami-08c757228751c5335", // EU (Paris).
	"me-south-1":     "ami-051274f257aba97f9", // Middle East (Bahrain).
	"sa-east-1":      "ami-077d5d3682940b34a", // South America (Sao Paulo).
	"us-east-1":      "ami-085925f297f89fce1", // US East (N. Virginia).
	"us-east-2":      "ami-07c1207a9d40bc3bd", // US East (Ohio).
	"us-west-1":      "ami-0f56279347d2fa43e", // US West (N. California).
	"us-west-2":      "ami-003634241a8fcdec0", // US West (Oregon).

	// AWS GovCloud (US) partition
	"us-gov-west-1": "ami-adecdbcc", // GovCloud (US-West)
	"us-gov-east-1": "ami-c29975b3", // GovCloud (US-East)
}
