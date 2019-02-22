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

package cluster

import (
	"fmt"
	"time"

	"github.com/banzaicloud/pipeline/model"
	pkgAuth "github.com/banzaicloud/pipeline/pkg/auth"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

const unknownLocation = "unknown"

// TableName constants
const (
	clustersTableName = "clusters"
)

// ClusterModel describes the common cluster model.
type ClusterModel struct {
	ID  pkgCluster.ClusterID `gorm:"primary_key"`
	UID string               `gorm:"unique_index:idx_uid"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `gorm:"unique_index:idx_unique_id" sql:"index"`
	CreatedBy pkgAuth.UserID

	Name           string `gorm:"unique_index:idx_unique_id"`
	Location       string
	Cloud          string
	Distribution   pkgCluster.DistributionID
	OrganizationID uint `gorm:"unique_index:idx_unique_id"`
	SecretID       string
	ConfigSecretID string
	SSHSecretID    string
	Status         string
	RbacEnabled    bool
	Monitoring     bool
	Logging        bool
	ServiceMesh    bool
	SecurityScan   bool
	StatusMessage  string             `sql:"type:text;"`
	ScaleOptions   model.ScaleOptions `gorm:"foreignkey:ClusterID"`
}

const InstanceTypeSeparator = " "

// TableName changes the default table name.
func (ClusterModel) TableName() string {
	return clustersTableName
}

func (m *ClusterModel) BeforeCreate() (err error) {
	if m.UID == "" {
		m.UID = uuid.Must(uuid.NewV4()).String()
	}

	return
}

// AfterFind converts Location field(s) to unknown if they are empty.
func (m *ClusterModel) AfterFind() error {
	if len(m.Location) == 0 {
		m.Location = unknownLocation
	}

	return nil
}

// String method prints formatted cluster fields.
func (m ClusterModel) String() string {
	return fmt.Sprintf("Id: %d, Creation date: %s, Cloud: %s, Distribution: %s", m.ID, m.CreatedAt, m.Cloud, m.Distribution)
}

// BeforeDelete should not be declared on this model.
// TODO: please move this to the cluster delete flow
// this should not have been added here in the first place!!!!!!!
func (m ClusterModel) BeforeDelete(tx *gorm.DB) (err error) {
	logger := log.WithFields(logrus.Fields{"organization": m.OrganizationID, "cluster": m.ID})

	logger.Info("Delete unused cluster secrets")
	if err := secret.Store.DeleteByClusterUID(m.OrganizationID, m.UID); err != nil {
		logger.Errorf("Error during deleting secret: %s", err.Error())
	}

	return
}
