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
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/ark/api"
)

// Service describes a service for every ARK related sync operations
type Service struct {
	clusterManager      api.ClusterManager
	bucketSyncInterval  time.Duration
	restoreSyncInterval time.Duration
	backupSyncInterval  time.Duration
}

// NewSyncService creates and initializes a Service
func NewSyncService(
	ClusterManager api.ClusterManager,
	BucketSyncInterval time.Duration,
	RestoreSyncInterval time.Duration,
	BackupSyncInterval time.Duration,
) *Service {

	return &Service{
		clusterManager:      ClusterManager,
		bucketSyncInterval:  BucketSyncInterval,
		restoreSyncInterval: RestoreSyncInterval,
		backupSyncInterval:  BackupSyncInterval,
	}
}

// Run runs every ARK related sync services
func (s *Service) Run(context context.Context, db *gorm.DB, logger logrus.FieldLogger) {

	var wg sync.WaitGroup

	// buckets
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.syncRegisteredBucketsLoop(context, db, logger, s.bucketSyncInterval)
	}()

	// restores
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.syncRestoresLoop(context, db, logger, s.restoreSyncInterval)
	}()

	// backups
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.syncBackupsLoop(context, db, logger, s.backupSyncInterval)
	}()

	wg.Wait()
}

func (s *Service) syncRegisteredBucketsLoop(
	ctx context.Context,
	db *gorm.DB,
	logger logrus.FieldLogger,
	interval time.Duration,
) {

	logger.WithField("interval", interval.String()).Debug("syncing backups from buckets")
	go s.syncRegisteredBuckets(db, logger)
	ticker := time.NewTicker(interval)
	func() {
		for {
			select {
			case <-ticker.C:
				logger.WithField("interval", interval.String()).Debug("syncing backups from buckets")
				s.syncRegisteredBuckets(db, logger)
			case <-ctx.Done():
				logger.Debug("closing ticker")
				ticker.Stop()
				return
			}
		}
	}()
}

func (s *Service) syncRegisteredBuckets(db *gorm.DB, logger logrus.FieldLogger) error {

	var orgs []*auth.Organization
	err := db.Find(&orgs).Error
	if err != nil {
		return err
	}

	for _, org := range orgs {
		log := logger.WithField("orgID", org.ID).WithField("orgName", org.Name)
		log.Debug("syncing backups from buckets")
		syncer := NewBucketsSyncService(org, db, logger)
		err := syncer.SyncBackupsFromBuckets()
		if err != nil {
			log.Error(err)
		}
	}

	return nil
}

func (s *Service) syncRestoresLoop(
	ctx context.Context,
	db *gorm.DB,
	logger logrus.FieldLogger,
	interval time.Duration,
) {

	logger.WithField("interval", interval.String()).Debug("syncing restores")
	go s.syncRestores(db, logger)
	ticker := time.NewTicker(interval)
	func() {
		for {
			select {
			case <-ticker.C:
				logger.WithField("interval", interval.String()).Debug("syncing restores")
				s.syncRestores(db, logger)
			case <-ctx.Done():
				logger.Debug("closing ticker")
				ticker.Stop()
				return
			}
		}
	}()
}

func (s *Service) syncRestores(db *gorm.DB, logger logrus.FieldLogger) error {

	var orgs []*auth.Organization
	err := db.Find(&orgs).Error
	if err != nil {
		return err
	}

	for _, org := range orgs {
		log := logger.WithField("orgID", org.ID).WithField("orgName", org.Name)
		log.Debug("syncing restores")
		syncer := NewRestoresSyncService(org, db, logger)
		err := syncer.SyncRestores(s.clusterManager)
		if err != nil {
			log.Error(err)
		}
	}

	return nil
}

func (s *Service) syncBackupsLoop(
	ctx context.Context,
	db *gorm.DB,
	logger logrus.FieldLogger,
	interval time.Duration,
) {

	logger.WithField("interval", interval.String()).Debug("syncing backups for organizations")
	go s.syncBackups(db, logger)
	ticker := time.NewTicker(interval)
	func() {
		for {
			select {
			case <-ticker.C:
				logger.WithField("interval", interval.String()).Debug("syncing backups for organizations")
				s.syncBackups(db, logger)
			case <-ctx.Done():
				logger.Debug("closing ticker")
				ticker.Stop()
				return
			}
		}
	}()
}

func (s *Service) syncBackups(db *gorm.DB, logger logrus.FieldLogger) error {

	var orgs []*auth.Organization
	err := db.Find(&orgs).Error
	if err != nil {
		return err
	}

	for _, org := range orgs {
		log := logger.WithField("orgID", org.ID).WithField("orgName", org.Name)
		log.Debug("syncing backups")
		syncer := NewBackupsSyncService(org, db, log)
		err := syncer.SyncBackups(s.clusterManager)
		if err != nil {
			log.Error(err)
		}
	}

	return nil
}
