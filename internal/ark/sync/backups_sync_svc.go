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

package sync

import (
	"context"

	"github.com/goph/emperror"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/ark"
	"github.com/banzaicloud/pipeline/internal/ark/api"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
)

// BackupsSyncService is for syncing backups between Pipeline DB and ARK for an Org
type BackupsSyncService struct {
	org    *auth.Organization
	db     *gorm.DB
	logger logrus.FieldLogger

	backupsSvc *ark.BackupsService
	bucketsSvc *ark.BucketsService
}

// NewBackupsSyncService returns an initialized BackupsSyncService
func NewBackupsSyncService(org *auth.Organization, db *gorm.DB, logger logrus.FieldLogger) *BackupsSyncService {

	s := &BackupsSyncService{
		org:    org,
		db:     db,
		logger: logger,
	}

	s.backupsSvc = ark.BackupsServiceFactory(s.org, s.db, s.logger)
	s.bucketsSvc = ark.BucketsServiceFactory(s.org, s.db, s.logger)

	return s
}

// SyncBackups syncs backups between Pipeline DB and ARK for every Cluster within the Org
func (s *BackupsSyncService) SyncBackups(clusterManager api.ClusterManager) error {

	// delete backups stored removed buckets
	s.logger.Debug("delete backups of removed buckets")
	err := ark.BackupsServiceFactory(s.org, s.db, s.logger).DeleteBackupsWithoutBucket()
	if err != nil {
		return err
	}

	clusters, err := clusterManager.GetClusters(context.Background(), s.org.ID)
	if err != nil {
		return err
	}

	// iterate through clusters and sync backups for each of them
	for _, cluster := range clusters {
		log := s.logger.WithField("clusterID", cluster.GetID())

		status, err := cluster.GetStatus()
		if err != nil {
			log.Error(emperror.Wrap(err, "could not get cluster status"))
			continue
		}

		if status.Status == pkgCluster.Deleting {
			continue
		}

		log.Debug("syncing backups for cluster")
		err = s.SyncBackupsForCluster(cluster)
		if err != nil && errors.Cause(err) != gorm.ErrRecordNotFound {
			log.Error(err)
		}
	}

	return nil
}

// SyncBackupsForCluster syncs backups between Pipeline DB and ARK for a Cluster
func (s *BackupsSyncService) SyncBackupsForCluster(cluster api.Cluster) error {
	deploymentsSvc := ark.DeploymentsServiceFactory(s.org, cluster, s.db, s.logger)

	deployment, err := deploymentsSvc.GetActiveDeployment()
	if err != nil {
		return emperror.Wrap(err, "could not get active deployment")
	}

	if deployment.RestoreMode == true {
		return nil
	}

	bucket, err := s.bucketsSvc.GetByID(deployment.BucketID)
	if err != nil {
		return emperror.Wrap(err, "could not get bucket by id")
	}

	client, err := deploymentsSvc.GetClient()
	if err != nil {
		return emperror.Wrap(err, "could not get ark client")
	}

	var listOptions metav1.ListOptions
	backups, err := client.ListBackups(listOptions)
	if err != nil {
		return emperror.Wrap(err, "could not list backups")
	}

	for _, backup := range backups.Items {
		log := s.logger.WithField("backup", backup.Name)
		req := &api.PersistBackupRequest{
			BucketID:     bucket.ID,
			Backup:       &backup,
			DeploymentID: bucket.DeploymentID,
			ClusterID:    bucket.ClusterID,
			Distribution: bucket.ClusterDistribution,
			Cloud:        bucket.ClusterCloud,
		}

		persitedBackup, err := s.backupsSvc.FindByPersistRequest(req)
		if err == gorm.ErrRecordNotFound {
			err = nil
		}
		if err != nil {
			log.Warning(err.Error())
			err = nil
			continue
		}

		if persitedBackup != nil && persitedBackup.ContentChecked != true && backup.Status.Phase == "Completed" {
			nodes, err := s.bucketsSvc.GetNodesFromBackupContents(bucket, backup.Name)
			if err != nil {
				log.Warning(err.Error())
				err = nil
				continue
			}
			req.ContentChecked = true
			req.Nodes = &nodes
			req.NodeCount = uint(len(nodes.Items))
			log.WithField("count", req.NodeCount).Debug("node count found")
		}

		_, err = s.backupsSvc.Persist(req)
		if err != nil {
			return emperror.Wrap(err, "could not persist backup")
		}

		log.Debug("backup synced")
	}

	return nil
}
