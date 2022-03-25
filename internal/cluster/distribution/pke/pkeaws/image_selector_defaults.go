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
// nolint: gochecknoglobals
var defaultImages = RegionMapImageSelector{
	// AWS partition
	"af-south-1":     "ami-063a57e81279c601b", // Africa (Cape Town).
	"ap-east-1":      "ami-c790d6b6",          // Asia Pacific (Hong Kong).
	"ap-northeast-1": "ami-0278fe6949f6b1a06", // Asia Pacific (Tokyo).
	"ap-northeast-2": "ami-00edfb46b107f643c", // Asia Pacific (Seoul).
	"ap-southeast-1": "ami-0f7719e8b7ba25c61", // Asia Pacific (Singapore).
	"ap-southeast-2": "ami-04fcc97b5f6edcd89", // Asia Pacific (Sydney).
	"ap-south-1":     "ami-0b44050b2d893d5f7", // Asia Pacific (Mumbai).
	"ca-central-1":   "ami-0edd51cc29813e254", // Canada (Central).
	"eu-central-1":   "ami-0e342d72b12109f91", // EU (Frankfurt).
	"eu-north-1":     "ami-050981837962d44ac", // EU (Stockholm).
	"eu-south-1":     "ami-027305c8710c4e8b5", // EU (Milan).
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
	"us-gov-east-1": "ami-c29975b3", // GovCloud (US-East)
	"us-gov-west-1": "ami-adecdbcc", // GovCloud (US-West)
}

// DefaultImages returns an image selector that returns fallback images if no other images are found.
// These are plain images, so PKE will be installed during the node bootstrapping.
func DefaultImages() ImageSelector {
	return defaultImages
}

// GPU PKE image: AWS Deep Learning Base AMI (Ubuntu 18.04)
// These are plain images, so PKE will be installed during the node bootstrapping.
// nolint: gochecknoglobals
var gpuImages = RegionMapImageSelector{
	// AWS partition
	"af-south-1":     "ami-041d491b1171625dc", // Africa (Cape Town).
	"ap-east-1":      "ami-0d244611e0ce5bbd9", // Asia Pacific (Hong Kong).
	"ap-northeast-1": "ami-0ee9352789b7c2121", // Asia Pacific (Tokyo).
	"ap-northeast-2": "ami-0636c2bd128d52753", // Asia Pacific (Seoul).
	"ap-southeast-1": "ami-0e5f6c7e6dc310278", // Asia Pacific (Singapore).
	"ap-southeast-2": "ami-02078f2564759a29b", // Asia Pacific (Sydney).
	"ap-south-1":     "ami-08a229ebcc1d01384", // Asia Pacific (Mumbai).
	"ca-central-1":   "ami-0a62e0ff8980ed692", // Canada (Central).
	"eu-central-1":   "ami-0b82b5c8831026cb6", // EU (Frankfurt).
	"eu-north-1":     "ami-0e2f4c4ff4a41934f", // EU (Stockholm).
	"eu-south-1":     "ami-0b4c48d24320b9326", // EU (Milan).
	"eu-west-1":      "ami-08265aab76b9d652d", // EU (Ireland).
	"eu-west-2":      "ami-0e37cd509765c681c", // EU (London).
	"eu-west-3":      "ami-029b88456846a0665", // EU (Paris).
	"me-south-1":     "ami-0884d458b0db56326", // Middle East (Bahrain).
	"sa-east-1":      "ami-016ee22fde76e0bbf", // South America (Sao Paulo).
	"us-east-1":      "ami-01be25d442771b889", // US East (N. Virginia).
	"us-east-2":      "ami-011f0f3a479c45fd9", // US East (Ohio).
	"us-west-1":      "ami-00469f2145d2161dc", // US West (N. California).
	"us-west-2":      "ami-0ff95feb4a4fabaf7", // US West (Oregon).
}

// GPUImages returns an image selector that returns GPU accelerated images.
// These are plain images, so PKE will be installed during the node bootstrapping.
func GPUImages() ImageSelector {
	return gpuImages
}
