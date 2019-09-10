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

package securityscan

//featureSpec security scan cluster feature specific specification
type featureSpec struct {
	CustomAnchore    anchoreSpec   `json:"customAnchore"`
	Policy           policySpec    `json:"policy"`
	ReleaseWhiteList []releaseSpec `json:"releaseWhiteList"`
}

type anchoreSpec struct {
	Enabled  bool   `json:"enabled"`
	Url      string `json:"url"`
	SecretID string `json:"secretId"`
}

type policySpec struct {
	PolicyID string `json:"policyId"`
}

type releaseSpec struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
	Regexp string `json:"regexp"`
}

type webHookConfigSpec struct {
	Enabled    bool     `json:"enabled"`
	Selector   string   `json:"selector"`
	Namespaces []string `json:"namespaces"`
}
