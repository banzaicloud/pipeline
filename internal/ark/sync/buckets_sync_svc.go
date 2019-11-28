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
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/ark"
	"github.com/banzaicloud/pipeline/internal/ark/api"
	"github.com/banzaicloud/pipeline/src/auth"
)

// BucketsSyncService is for syncing backups from object store buckets
type BucketsSyncService struct {
	org *auth.Organization

	bucketsSvc *ark.BucketsService
	backupsSvc *ark.BackupsService

	logger logrus.FieldLogger
}

// NewBucketsSyncService returns an initialized BucketsSyncService
func NewBucketsSyncService(
	org *auth.Organization,
	db *gorm.DB,
	logger logrus.FieldLogger,
) *BucketsSyncService {

	return &BucketsSyncService{
		org:        org,
		bucketsSvc: ark.BucketsServiceFactory(org, db, logger),
		backupsSvc: ark.BackupsServiceFactory(org, db, logger),

		logger: logger,
	}
}

// SyncBackupsFromBuckets syncs backups from object store buckets for ARK backup bucket for within the organization
func (s *BucketsSyncService) SyncBackupsFromBuckets() error {

	buckets, err := s.bucketsSvc.List()
	if err != nil {
		return err
	}

	for _, bucket := range buckets {
		log := s.logger.WithField("bucket", bucket.Name)
		log.Debug("syncing backups from bucket")
		backupIDS, err := s.syncBackupsFromBucket(bucket)
		if err != nil {
			log.Warning(err.Error())
		}

		log.Debug("removing deleted backups from database")
		err = s.backupsSvc.DeleteNonExistingBackupsByBucketAndKeys(bucket.ID, backupIDS)
		if err != nil {
			log.Error(err.Error())
			continue
		}
	}

	return nil
}

func (s *BucketsSyncService) syncBackupsFromBucket(bucket *api.Bucket) (backupIDS []int, err error) {

	backupIDS = make([]int, 0)

	log := s.logger.WithField("bucket", bucket.Name)

	backups, err := s.bucketsSvc.GetBackupsFromObjectStore(bucket)
	if err != nil {
		return
	}

	for _, backup := range backups {
		log = log.WithField("backup", backup.Name)
		req := &api.PersistBackupRequest{
			BucketID:     bucket.ID,
			Backup:       backup,
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

		syncedBackup, err := s.backupsSvc.Persist(req)
		if err != nil {
			return backupIDS, err
		}

		log.Debug("backup synced")

		backupIDS = append(backupIDS, int(syncedBackup.ID))
	}

	return backupIDS, err
}
