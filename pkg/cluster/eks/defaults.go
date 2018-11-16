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

// ### [ Constants to EKS cluster default values ] ### //
const (
	DefaultInstanceType = "m4.xlarge"
	DefaultSpotPrice    = "0.0" // 0 spot price stands for on-demand instances
	DefaultRegion       = UsWest2
)

// DefaultImages in each supported location in EC2 (from https://docs.aws.amazon.com/eks/latest/userguide/launch-workers.html)
var DefaultImages = map[string]string{
	UsEast1: "ami-0a0b913ef3249b655",
	UsEast2: "ami-0958a76db2d150238",
	UsWest2: "ami-0f54a2f7d2e9c88b3",
	EuWest1: "ami-00c3b2d35bddd4f5c",
}

// EC2 regions
const (
	UsEast1 = "us-east-1"
	UsEast2 = "us-east-2"
	UsWest2 = "us-west-2"
	EuWest1 = "eu-west-1"
)
