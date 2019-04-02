// Copyright © 2019 Banzai Cloud
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

package api

// Kubernetes defines properties of K8s
type Kubernetes struct {
	CRI     CRI     `json:"cri" yaml:"cri"`
	Network Network `json:"network" yaml:"network"`
	RBAC    bool    `json:"rbac" yaml:"rbac"`
	Version string  `json:"version" yaml:"version" binding:"required"`
}

// CRI defines properties of the container runtime interface
type CRI struct {
	Runtime       string                 `json:"runtime" yaml:"runtime"`
	RuntimeConfig map[string]interface{} `json:"runtimeConfig" yaml:"runtimeConfig"`
}

// Network defines properties of the K8s network
type Network struct {
	PodCIDR        string                 `json:"podCIDR" yaml:"podCIDR"`
	Provider       string                 `json:"provider" yaml:"provider"`
	ProviderConfig map[string]interface{} `json:"providerConfig" yaml:"providerConfig"`
	ServiceCIDR    string                 `json:"serviceCIDR" yaml:"serviceCIDR"`
}
