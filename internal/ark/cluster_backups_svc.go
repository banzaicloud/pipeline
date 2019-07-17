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
	"emperror.dev/emperror"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/ark/api"
)

// ClusterBackupsService is for cluster backups related ARK functions
type ClusterBackupsService struct {
	*BackupsService
	deployments *DeploymentsService
	cluster     api.Cluster
}

// ClusterBackupsServiceFactory creates and returns an initialized ClusterBackupsService instance
func ClusterBackupsServiceFactory(
	org *auth.Organization,
	deployments *DeploymentsService,
	db *gorm.DB,
	logger logrus.FieldLogger,
) *ClusterBackupsService {

	repository := NewBackupsRepository(org, db, logger)
	backups := NewBackupsService(org, repository, logger)

	return NewClusterBackupsService(backups, deployments)
}

// NewClusterBackupsService creates and returns an initialized ClusterBackupsService instance
func NewClusterBackupsService(backups *BackupsService, deployments *DeploymentsService) *ClusterBackupsService {

	return &ClusterBackupsService{
		BackupsService: backups,
		deployments:    deployments,
		cluster:        deployments.GetCluster(),
	}
}

// DeleteByName deletes an ARK backup by name
func (s *ClusterBackupsService) DeleteByName(name string) error {

	_, err := s.deployments.GetActiveDeployment()
	if err != nil {
		return emperror.Wrap(err, "error getting active deployment")
	}

	client, err := s.deployments.GetClient()
	if err != nil {
		return emperror.Wrap(err, "error getting ark client")
	}

	backup, err := s.repository.FindOneByName(name)
	if err != nil {
		return emperror.Wrap(err, "backup not found")
	}

	if backup.Bucket.Deployment.ClusterID != s.cluster.GetID() {
		return errors.New("backup not available for delete")
	}

	err = s.repository.UpdateStatus(backup, "Deleting", "deleting backup...")
	if err != nil {
		return emperror.Wrap(err, "cannot update backup status")
	}

	err = client.CreateDeleteBackupRequestByName(backup.Name)
	if err != nil {
		return emperror.Wrap(err, "error during deleting backup")
	}

	return nil
}

// DeleteByID deletes an ARK backup by ID
func (s *ClusterBackupsService) DeleteByID(id uint) error {

	_, err := s.deployments.GetActiveDeployment()
	if err != nil {
		return emperror.Wrap(err, "error getting active deployment")
	}

	client, err := s.deployments.GetClient()
	if err != nil {
		return emperror.Wrap(err, "error getting ark client")
	}

	backup, err := s.repository.FindOneByID(id)
	if err != nil {
		return emperror.Wrap(err, "backup not found")
	}

	if backup.Bucket.Deployment.ClusterID != s.cluster.GetID() {
		return errors.New("backup not available for delete")
	}

	err = s.repository.UpdateStatus(backup, "Deleting", "deleting backup...")
	if err != nil {
		return emperror.Wrap(err, "cannot update backup status")
	}

	err = client.CreateDeleteBackupRequestByName(backup.Name)
	if err != nil {
		return emperror.Wrap(err, "error during deleting backup")
	}

	return nil
}

// Create creates and persists an ARK backup by a CreateBackupRequest
func (s *ClusterBackupsService) Create(req api.CreateBackupRequest) error {

	deployment, err := s.deployments.GetActiveDeployment()
	if err != nil {
		return emperror.Wrap(err, "error getting active deployment")
	}

	client, err := s.deployments.GetClient()
	if err != nil {
		return emperror.Wrap(err, "error getting ark client")
	}

	if req.Labels == nil {
		req.Labels = make(labels.Set)
	}
	req.Labels[api.LabelKeyDistribution] = string(s.cluster.GetDistribution())
	req.Labels[api.LabelKeyCloud] = s.cluster.GetCloud()

	backup, err := client.CreateBackup(req)
	if err != nil {
		return emperror.Wrap(err, "error creating backup")
	}

	if backup.Status.Phase == "" {
		backup.Status.Phase = "Creating"
	}

	preq := &api.PersistBackupRequest{
		BucketID: deployment.BucketID,

		Cloud:        s.cluster.GetCloud(),
		Distribution: s.cluster.GetDistribution(),

		ClusterID:    s.cluster.GetID(),
		DeploymentID: deployment.ID,

		Backup: backup,
	}
	_, err = s.Persist(preq)
	if err != nil {
		return emperror.Wrap(err, "error persisting backup")
	}

	return nil
}
