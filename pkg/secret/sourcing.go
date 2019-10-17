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

package secret

// SourcingMethod describes how an installed Secret should be sourced into a Pod in K8S
type SourcingMethod string

const (
	// EnvVar means the secret has to be sources an an env var
	EnvVar SourcingMethod = "env"
	// Volume means the secret has to be mounted an a volume
	Volume SourcingMethod = "volume"
)

// K8SSourceMeta describes which and how installed Secret should be sourced into a Pod in K8S
type K8SSourceMeta struct {
	Name     string         `json:"name"`
	Sourcing SourcingMethod `json:"sourcing"`
}
