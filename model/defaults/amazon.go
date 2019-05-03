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

package defaults

type AmazonNodePoolProfileBaseFields struct {
	ID           uint   `gorm:"primary_key"`
	InstanceType string `gorm:"default:'m4.xlarge'"`
	Name         string `gorm:"unique_index:idx_amazon_name_node_name"`
	NodeName     string `gorm:"unique_index:idx_amazon_name_node_name"`
	SpotPrice    string
	Autoscaling  bool `gorm:"default:false"`
	MinCount     int  `gorm:"default:1"`
	MaxCount     int  `gorm:"default:2"`
	Count        int  `gorm:"default:1"`
}
