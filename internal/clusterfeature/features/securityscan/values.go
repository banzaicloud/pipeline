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

import v1 "k8s.io/api/core/v1"

type SecurityScanChartValues struct {
	Anchore     AnchoreValues   `json:"externalAnchore"`
	Affinity    *v1.Affinity    `json:"affinity,omitempty"`
	Tolerations []v1.Toleration `json:"tolerations,omitempty"`
}

// AnchoreValues struct used to build chart values and to extract anchore data from secret values
type AnchoreValues struct {
	Host     string `json:"anchoreHost" mapstructure:"host"`
	User     string `json:"anchoreUser" mapstructure:"username"`
	Password string `json:"anchorePassword" mapstructure:"password"`
}
