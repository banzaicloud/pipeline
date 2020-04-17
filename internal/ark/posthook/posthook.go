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

package posthook

import (
	"time"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/banzaicloud/pipeline/internal/helm"

	"github.com/banzaicloud/pipeline/internal/ark"
	"github.com/banzaicloud/pipeline/internal/ark/api"
	"github.com/banzaicloud/pipeline/internal/ark/sync"
	"github.com/banzaicloud/pipeline/src/auth"
)

const (
	retrySleepSeconds    = 15
	restoredByLabelKey   = "restored-by"
	restoredByLabelValue = "pipeline"
)

// nolint: gochecknoglobals
var (
	nonRestorableNamespaces = []string{
		"kube-system",
	}
)

// RestoreFromBackup is a posthook for restoring a backup right after a new cluster is created
func RestoreFromBackup(
	params api.RestoreFromBackupParams,
	cluster api.Cluster,
	db *gorm.DB,
	logger logrus.FieldLogger,
	errorHandler emperror.Handler,
	waitTimeout time.Duration,
	helmService helm.UnifiedReleaser,
) error {
	org, err := auth.GetOrganizationById(cluster.GetOrganizationId())
	if err != nil {
		return err
	}

	svc := ark.NewARKService(org, cluster, db, logger)
	backupsSvc := svc.GetBackupsService()

	backup, err := backupsSvc.GetModelByID(params.BackupID)
	if err != nil {
		return err
	}

	err = svc.GetDeploymentsService().Deploy(helmService, &backup.Bucket, true)
	if err != nil {
		return err
	}

	labels := make(labels.Set)
	labels[restoredByLabelKey] = restoredByLabelValue

	restoresSvc := svc.GetRestoresService()
	restore, err := restoresSvc.Create(api.CreateRestoreRequest{
		BackupName: backup.Name,
		Labels:     labels,
		Options: api.RestoreOptions{
			ExcludedNamespaces: nonRestorableNamespaces,
		},
	})
	if err == nil {
		err = WaitingForRestoreToFinish(restoresSvc, sync.NewRestoresSyncService(org, db, logger), cluster, restore, logger, waitTimeout)
	}
	if err != nil {
		errorHandler.Handle(errors.WrapIf(err, "could not restore"))
	}

	err = svc.GetDeploymentsService().Remove(helmService)
	if err != nil {
		return err
	}

	return nil
}

// WaitingForRestoreToFinish waits until restoration process finishes
func WaitingForRestoreToFinish(restoresSvc *ark.RestoresService, restoresSyncSvc *sync.RestoresSyncService, cluster api.Cluster, restore *api.Restore, logger logrus.FieldLogger, waitTimeout time.Duration) error {
	retryAttempts := int(waitTimeout.Seconds() / retrySleepSeconds)

	for i := 0; i <= retryAttempts; i++ {
		err := restoresSyncSvc.SyncRestoresForCluster(cluster)
		if err != nil {
			return errors.WrapIfWithDetails(err, "could not sync restores for cluster", "cluster", cluster.GetName())
		}
		r, err := restoresSvc.GetByName(restore.Name)
		if err != nil {
			return errors.WrapIfWithDetails(err, "could not get restore by name", "restore", restore.Name, "cluster", cluster.GetName())
		}
		if r.Status == "Completed" {
			return nil
		}
		logger.WithFields(logrus.Fields{
			"status":       r.Status,
			"attempt":      i,
			"max-attempts": retryAttempts,
		}).Debug("restoration in progress")
		time.Sleep(time.Duration(retrySleepSeconds) * time.Second)
	}

	return errors.New("timeout during waiting for restoration to finish")
}
