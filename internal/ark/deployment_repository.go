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

package ark

import (
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/ark/api"
	"github.com/banzaicloud/pipeline/src/auth"
)

// DeploymentsRepository is a repository for storing ARK deployment models
type DeploymentsRepository struct {
	org     *auth.Organization
	cluster api.Cluster
	db      *gorm.DB
	logger  logrus.FieldLogger
}

// NewDeploymentsRepository creates and returns a DeploymentsRepository instance
func NewDeploymentsRepository(
	org *auth.Organization,
	cluster api.Cluster,
	db *gorm.DB,
	logger logrus.FieldLogger,
) *DeploymentsRepository {

	return &DeploymentsRepository{
		org:     org,
		cluster: cluster,
		db:      db,
		logger:  logger,
	}
}

// FindFirst gets the first ClusterBackupDeploymentsModel for a cluster (normally there must only be one per cluster)
func (s *DeploymentsRepository) FindFirst() (*ClusterBackupDeploymentsModel, error) {

	var deployment ClusterBackupDeploymentsModel
	err := s.db.Where(&ClusterBackupDeploymentsModel{
		ClusterID:      s.cluster.GetID(),
		OrganizationID: s.org.ID,
	}).First(&deployment).Error
	if err != nil {
		return nil, err
	}

	return &deployment, nil
}

// Persist creates and persists a ClusterBackupDeploymentsModel by a PersistDeploymentRequest
func (s *DeploymentsRepository) Persist(req *api.PersistDeploymentRequest) (*ClusterBackupDeploymentsModel, error) {

	deployment := &ClusterBackupDeploymentsModel{
		BucketID:    req.BucketID,
		RestoreMode: req.RestoreMode,
		Name:        req.Name,
		Namespace:   req.Namespace,

		Status:         "DEPLOYING",
		OrganizationID: s.org.ID,
		ClusterID:      s.cluster.GetID(),
	}

	return deployment, s.db.Save(deployment).Error
}

// Delete deletes a ClusterBackupDeploymentsModel
func (s *DeploymentsRepository) Delete(deployment *ClusterBackupDeploymentsModel) error {

	return s.db.Delete(&deployment).Error
}

// UpdateStatus updates the status of a ClusterBackupDeploymentsModel
func (s *DeploymentsRepository) UpdateStatus(deployment *ClusterBackupDeploymentsModel, status, message string) error {

	deployment.Status = status
	deployment.StatusMessage = message

	return s.db.Save(&deployment).Error
}
