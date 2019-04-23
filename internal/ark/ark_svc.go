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

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/ark/api"
)

// Service is for various ARK related management functions
type Service struct {
	cluster api.Cluster
	org     *auth.Organization
	db      *gorm.DB

	deploymentsSvc    *DeploymentsService
	bucketsSvc        *BucketsService
	backupsSvc        *BackupsService
	clusterBackupsSvc *ClusterBackupsService
	schedulesSvc      *SchedulesService
	restoresSvc       *RestoresService

	logger logrus.FieldLogger
}

// NewARKService returns a new ARKService instance
func NewARKService(org *auth.Organization, cluster api.Cluster, db *gorm.DB, logger logrus.FieldLogger) *Service {

	backups := BackupsServiceFactory(org, db, logger)
	buckets := BucketsServiceFactory(org, db, logger)
	deployments := DeploymentsServiceFactory(org, cluster, db, logger)
	schedules := SchedulesServiceFactory(deployments, logger)
	clusterBackups := ClusterBackupsServiceFactory(org, deployments, db, logger)
	restores := RestoresServiceFactory(org, deployments, db, logger)

	return &Service{
		org:               org,
		cluster:           cluster,
		bucketsSvc:        buckets,
		backupsSvc:        backups,
		clusterBackupsSvc: clusterBackups,
		deploymentsSvc:    deployments,
		schedulesSvc:      schedules,
		restoresSvc:       restores,
		logger:            logger,
		db:                db,
	}
}

// GetClusterBackupsService returns the initialized ClusterBackupsService
func (s *Service) GetClusterBackupsService() *ClusterBackupsService {
	return s.clusterBackupsSvc
}

// GetSchedulesService returns the initialized SchedulesService
func (s *Service) GetSchedulesService() *SchedulesService {
	return s.schedulesSvc
}

// GetBucketsService returns the initialized BucketsService
func (s *Service) GetBucketsService() *BucketsService {
	return s.bucketsSvc
}

// GetDeploymentsService returns the initialized DeploymentsService
func (s *Service) GetDeploymentsService() *DeploymentsService {
	return s.deploymentsSvc
}

// GetBackupsService returns the initialized BackupsService
func (s *Service) GetBackupsService() *BackupsService {
	return s.backupsSvc
}

// GetRestoresService returns the initialized RestoresService
func (s *Service) GetRestoresService() *RestoresService {
	return s.restoresSvc
}

// GetDB returns the DB instance used in the service
func (s *Service) GetDB() *gorm.DB {
	return s.db
}

// GetCluster returns the cluster used in the service
func (s *Service) GetCluster() api.Cluster {
	return s.cluster
}

// GetOrganization returns the organization used in the service
func (s *Service) GetOrganization() *auth.Organization {
	return s.org
}
