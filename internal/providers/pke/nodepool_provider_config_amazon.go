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

package pke

type NodePoolProviderConfigAmazon struct {
	AutoScalingGroup struct {
		Name                    string  `yaml:"name"`
		Image                   string  `yaml:"image"`
		Zones                   Zones   `yaml:"zones"`
		InstanceType            string  `yaml:"instanceType"`
		LaunchConfigurationName string  `yaml:"launchConfigurationName"`
		LaunchTemplate          string  `yaml:"launchTemplate"`
		VPCID                   string  `yaml:"vpcID"`
		SecurityGroupID         string  `yaml:"securityGroupID"`
		Subnets                 Subnets `yaml:"subnets"`
		Tags                    Tags    `yaml:"tags"`
		Size                    struct {
			Min     int `yaml:"min"`
			Max     int `yaml:"max"`
			Desired int `yaml:"desired"`
		} `yaml:"size"`
		SpotPrice string `yaml:"spotPrice"`
	} `yaml:"autoScalingGroup"`
}

type Zones []Zone
type Zone string

type Subnets []Subnet
type Subnet string

type Tags map[string]string
