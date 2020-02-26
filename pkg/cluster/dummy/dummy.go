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

package dummy

// CreateClusterDummy describes Pipeline's Dummy fields of a CreateCluster request
type CreateClusterDummy struct {
	Node *Node `json:"node,omitempty"`
}

// Node describes Dummy's node fields of a CreateCluster/Update request
type Node struct {
	KubernetesVersion string `json:"kubernetesVersion" yaml:"kubernetesVersion"`
	Count             int    `json:"count" yaml:"count"`
}

// UpdateClusterDummy describes Dummy's node fields of an UpdateCluster request
type UpdateClusterDummy struct {
	Node *Node `json:"node,omitempty"`
}

// Validate validates cluster create request
func (d *CreateClusterDummy) Validate() error {
	if d.Node == nil {
		d.Node = &Node{
			KubernetesVersion: "DummyKubernetesVersion",
			Count:             1,
		}
	}

	return nil
}

// Validate validates the update request
func (r *UpdateClusterDummy) Validate() error {
	if r.Node == nil {
		r.Node = &Node{
			KubernetesVersion: "DummyKubernetesVersion",
			Count:             1,
		}
	}
	return nil
}
