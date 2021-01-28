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

package model

import (
	"bytes"
	"database/sql/driver"
	"fmt"
	"time"

	"emperror.dev/errors"
	"github.com/gofrs/uuid"

	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter/clustermodel"
	"github.com/banzaicloud/pipeline/internal/database/sql/json"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/providers/azure/azureadapter"
	"github.com/banzaicloud/pipeline/internal/providers/kubernetes/kubernetesadapter"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
)

const unknown = "unknown"

// ClusterModel describes the common cluster model
// Note: this model is being moved to github.com/banzaicloud/pipeline/internal/cluster/clusteradapter.ClusterModel
type ClusterModel struct {
	ID             uint   `gorm:"primary_key"`
	UID            string `gorm:"unique_index:idx_clusters_uid"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time `gorm:"unique_index:idx_clusters_unique_id" sql:"index"`
	StartedAt      *time.Time
	Name           string `gorm:"unique_index:idx_clusters_unique_id"`
	Location       string
	Cloud          string
	Distribution   string
	OrganizationId uint `gorm:"unique_index:idx_clusters_unique_id"`
	SecretId       string
	ConfigSecretId string
	SshSecretId    string
	Status         string
	RbacEnabled    bool
	ScaleOptions   clustermodel.ScaleOptions                `gorm:"foreignkey:ClusterID"`
	StatusMessage  string                                   `sql:"type:text;"`
	AKS            azureadapter.AKSClusterModel             `gorm:"foreignkey:ID"`
	Kubernetes     kubernetesadapter.KubernetesClusterModel `gorm:"foreignkey:ID"`
	CreatedBy      uint
	Tags           ClusterTags `gorm:"type:json"`
}

type ClusterTags map[string]string

// TableName sets ClusterModel's table name
func (ClusterModel) TableName() string {
	return "clusters"
}

func (cs *ClusterModel) BeforeCreate() (err error) {
	if cs.UID == "" {
		cs.UID = uuid.Must(uuid.NewV4()).String()
	}

	return
}

// AfterFind converts metadata json string into map in case of Kubernetes and sets NodeInstanceType and/or Location field(s)
// to unknown if they are empty
func (cs *ClusterModel) AfterFind() error {
	if len(cs.Location) == 0 {
		cs.Location = unknown
	}

	return nil
}

// Save the cluster to DB
func (cs *ClusterModel) Save() error {
	db := global.DB()
	err := db.Save(&cs).Error
	if err != nil {
		return err
	}
	return nil
}

// Delete cluster from DB
func (cs *ClusterModel) Delete() error {
	db := global.DB()
	return db.Delete(&cs).Error
}

// String method prints formatted cluster fields
func (cs *ClusterModel) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("Id: %d, Creation date: %s, Cloud: %s, Distribution: %s, ", cs.ID, cs.CreatedAt, cs.Cloud, cs.Distribution))

	switch cs.Distribution {
	case pkgCluster.AKS:
		// Write AKS
		buffer.WriteString(fmt.Sprintf("NodePools: %v, Kubernetes version: %s",
			cs.AKS.NodePools,
			cs.AKS.KubernetesVersion))
	case pkgCluster.Kubernetes:
		buffer.WriteString(fmt.Sprintf("Metadata: %#v", cs.Kubernetes.Metadata))
	}

	return buffer.String()
}

// UpdateStatus updates the model's status and status message in database
func (cs *ClusterModel) UpdateStatus(status, statusMessage string) error {
	if cs.Status == status && cs.StatusMessage == statusMessage {
		return nil
	}

	if cs.ID != 0 {
		// Record status change to history before modifying the actual status.
		// If setting/saving the actual status doesn't succeed somehow, at least we can reconstruct it from history (i.e. event sourcing).
		statusHistory := clustermodel.StatusHistoryModel{
			ClusterID:   cs.ID,
			ClusterName: cs.Name,

			FromStatus:        cs.Status,
			FromStatusMessage: cs.StatusMessage,
			ToStatus:          status,
			ToStatusMessage:   statusMessage,
		}

		if err := global.DB().Save(&statusHistory).Error; err != nil {
			return errors.Wrap(err, "failed to record cluster status change to history")
		}
	}

	if cs.Status == pkgCluster.Creating && (cs.Status == pkgCluster.Running || cs.Status == pkgCluster.Warning) {
		now := time.Now()
		cs.StartedAt = &now
	}
	cs.Status = status
	cs.StatusMessage = statusMessage

	if err := cs.Save(); err != nil {
		return errors.Wrap(err, "failed to update cluster status")
	}

	return nil
}

// UpdateConfigSecret updates the model's config secret id in database
func (cs *ClusterModel) UpdateConfigSecret(configSecretId string) error {
	cs.ConfigSecretId = configSecretId
	return cs.Save()
}

// UpdateSshSecret updates the model's ssh secret id in database
func (cs *ClusterModel) UpdateSshSecret(sshSecretId string) error {
	cs.SshSecretId = sshSecretId
	return cs.Save()
}

func (fs *ClusterTags) Scan(src interface{}) error {
	return json.Scan(src, fs)
}

func (fs ClusterTags) Value() (driver.Value, error) {
	return json.Value(fs)
}
