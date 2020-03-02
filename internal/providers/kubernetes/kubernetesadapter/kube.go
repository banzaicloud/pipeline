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

package kubernetesadapter

import (
	"encoding/json"
)

// KubernetesClusterModel describes the build your own cluster model
type KubernetesClusterModel struct {
	ID          uint              `gorm:"primary_key"`
	Metadata    map[string]string `gorm:"-"`
	MetadataRaw []byte            `gorm:"meta_data"`
}

// BeforeSave converts the metadata into a json string in case of Kubernetes
func (cs *KubernetesClusterModel) BeforeSave() (err error) {
	cs.MetadataRaw, err = json.Marshal(cs.Metadata)
	return
}

// AfterFind converts the metadata json string to a map in case of Kubernetes
func (cs *KubernetesClusterModel) AfterFind() error {
	if len(cs.MetadataRaw) != 0 {
		return json.Unmarshal(cs.MetadataRaw, &cs.Metadata)
	}
	return nil
}

// TableName sets the KubernetesClusterModel's table name
func (KubernetesClusterModel) TableName() string {
	return "kubernetes_clusters"
}
