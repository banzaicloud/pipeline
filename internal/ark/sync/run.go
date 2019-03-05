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
	"time"

	"github.com/banzaicloud/pipeline/internal/ark/api"
	"github.com/goph/emperror"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

// RunSyncServices runs ARK sync services
func RunSyncServices(
	context context.Context,
	db *gorm.DB,
	clusterManager api.ClusterManager,
	logger logrus.FieldLogger,
	errorHandler emperror.Handler,
	bucketSyncInterval, restoreSyncInterval, backupSyncInterval time.Duration,
) {
	if bucketSyncInterval.Seconds() < 1 {
		logger.WithField("interval", bucketSyncInterval.Seconds()).Error("invalid bucket sync interval")
		return
	}
	if restoreSyncInterval.Seconds() < 1 {
		logger.WithField("interval", restoreSyncInterval.Seconds()).Error("invalid restore sync interval")
		return
	}
	if backupSyncInterval.Seconds() < 1 {
		logger.WithField("interval", backupSyncInterval.Seconds()).Error("invalid backup sync interval")
		return
	}

	logger.WithFields(logrus.Fields{
		"bucket-sync-interval":  bucketSyncInterval,
		"restore-sync-interval": restoreSyncInterval,
		"backup-sync-interval":  backupSyncInterval,
	}).Info("ARK synchronisation starting")

	svc := NewSyncService(
		clusterManager,
		bucketSyncInterval,
		restoreSyncInterval,
		backupSyncInterval,
	)

	svc.Run(context, db, logger)
}
