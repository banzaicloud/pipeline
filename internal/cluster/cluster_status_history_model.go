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

package cluster

import (
	"time"
)

const (
	clusterStatusHistoryTableName = "cluster_status_history"
)

// StatusHistoryModel records the status transitions of a cluster and stores it in a database.
type StatusHistoryModel struct {
	ID uint `gorm:"primary_key"`

	ClusterID   uint      `gorm:"not null;index"`
	ClusterName string    `gorm:"not null"`
	CreatedAt   time.Time `gorm:"not null"`

	FromStatus        string `gorm:"not null"`
	FromStatusMessage string `sql:"type:text;" gorm:"not null"`
	ToStatus          string `gorm:"not null"`
	ToStatusMessage   string `sql:"type:text;" gorm:"not null"`
}

// TableName changes the default table name.
func (StatusHistoryModel) TableName() string {
	return clusterStatusHistoryTableName
}
