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

package clustermodel

// ClusterTag describes custom tags for a cluster
type ClusterTag struct {
	ID        uint   `gorm:"primary_key"`
	ClusterID uint   `gorm:"unique_index:idx_cluster_tags_unique_id"`
	Key       string `gorm:"unique_index:idx_cluster_tags_unique_id"`
	Value     string
}

// TableName changes the default table name.
func (ClusterTag) TableName() string {
	return "cluster_tags"
}
