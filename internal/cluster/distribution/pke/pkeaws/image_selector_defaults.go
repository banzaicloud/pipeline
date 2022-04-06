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

// Default PKE image: Ubuntu 20.04
// These are plain images, so PKE will be installed during the node bootstrapping.
// nolint: gochecknoglobals
var defaultImages = RegionMapImageSelector{
	// AWS partition
	"af-south-1":     "ami-04402921e3d389a19", // Africa (Cape Town).
	"ap-east-1":      "ami-02ff31ab125adcde7", // Asia Pacific (Hong Kong).
	"ap-northeast-1": "ami-04110a75416d1c646", // Asia Pacific (Tokyo).
	"ap-northeast-2": "ami-07cc5503f90076a9a", // Asia Pacific (Seoul).
	"ap-northeast-3": "ami-0494bd48a487f67fd", // Asia Pacific (Osaka).
	"ap-southeast-1": "ami-0e91c34d3c7ee6fae", // Asia Pacific (Singapore).
	"ap-southeast-2": "ami-0b72e846931519ea3", // Asia Pacific (Sydney).
	"ap-southeast-3": "ami-044c8c46473270be6", // Asia Pacific (Jakarta).
	"ap-south-1":     "ami-08f7245e4682ed599", // Asia Pacific (Mumbai).
	"ca-central-1":   "ami-03441c48a85de729c", // Canada (Central).
	"eu-central-1":   "ami-0498a49a15494604f", // EU (Frankfurt).
	"eu-north-1":     "ami-04e4f9b92505045dd", // EU (Stockholm).
	"eu-south-1":     "ami-032397668a37f94af", // EU (Milan).
	"eu-west-1":      "ami-0ef38d2cfb7fd2d03", // EU (Ireland).
	"eu-west-2":      "ami-01d912b5940be07a5", // EU (London).
	"eu-west-3":      "ami-01cec60ec758ad370", // EU (Paris).
	"me-south-1":     "ami-0170daa4ae2ca72ba", // Middle East (Bahrain).
	"sa-east-1":      "ami-05c7374d9286c47e8", // South America (Sao Paulo).
	"us-east-1":      "ami-01896de1f162f0ab7", // US East (N. Virginia).
	"us-east-2":      "ami-045137e8d34668746", // US East (Ohio).
	"us-west-1":      "ami-0189702ff9c0b592f", // US West (N. California).
	"us-west-2":      "ami-0b0e59a09e7f4059f", // US West (Oregon).

	// AWS GovCloud (US) partition
	"us-gov-east-1": "ami-002a7501b87593bdd", // GovCloud (US-East)
	"us-gov-west-1": "ami-0b152eed9cb83f2bd", // GovCloud (US-West)
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
