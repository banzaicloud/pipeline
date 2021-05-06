// Copyright © 2021 Banzai Cloud
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

package velero

import (
	"context"
	"fmt"
	"time"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"go.uber.org/cadence"
	"go.uber.org/cadence/activity"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/banzaicloud/pipeline/internal/ark"
	"github.com/banzaicloud/pipeline/internal/ark/api"
	"github.com/banzaicloud/pipeline/internal/ark/sync"
	"github.com/banzaicloud/pipeline/internal/cmd"
	"github.com/banzaicloud/pipeline/internal/global"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/cluster"
)

const (
	retrySleepSeconds      = 15
	restoredByLabelKey     = "restored-by"
	restoredByLabelValue   = "pipeline"
	ErrReasonRestoreFailed = "BACKUP_RESTORE_FAILED"
)

// nolint: gochecknoglobals
var (
	nonRestorableNamespaces = []string{
		"kube-system",
		"pipeline-system",
	}
)

// ClusterManager interface to access clusters.
type ClusterManager interface {
	GetClusterByIDOnly(ctx context.Context, clusterID uint) (cluster.CommonCluster, error)
}

type HelmService interface {
	InstallDeployment(
		ctx context.Context,
		clusterID uint,
		namespace string,
		chartName string,
		releaseName string,
		values []byte,
		chartVersion string,
		wait bool,
	) error

	DeleteDeployment(ctx context.Context, clusterID uint, releaseName, namespace string) error
}

type RestoreBackupActivityInput struct {
	ClusterID           uint
	RestoreBackupParams pkgCluster.RestoreFromBackupParams
}

type RestoreBackupActivity struct {
	manager                ClusterManager
	helmService            HelmService
	db                     *gorm.DB
	disasterRecoveryConfig cmd.ClusterDisasterRecoveryConfig
}

func NewRestoreBackupActivity(manager ClusterManager, helmService HelmService, db *gorm.DB, disasterRecoveryConfig cmd.ClusterDisasterRecoveryConfig) *RestoreBackupActivity {
	return &RestoreBackupActivity{
		manager:                manager,
		helmService:            helmService,
		db:                     db,
		disasterRecoveryConfig: disasterRecoveryConfig,
	}
}

func (a RestoreBackupActivity) Execute(ctx context.Context, input RestoreBackupActivityInput) error {
	if !a.disasterRecoveryConfig.Enabled {
		return nil
	}

	cluster, err := a.manager.GetClusterByIDOnly(ctx, input.ClusterID)
	if err != nil {
		return err
	}

	org, err := auth.GetOrganizationById(cluster.GetOrganizationId())
	if err != nil {
		return err
	}

	info := activity.GetInfo(ctx)
	logrusLogger := global.LogrusLogger()
	logrusLogger.WithField("clusterID", input.ClusterID).
		WithField("workflowID", info.WorkflowExecution.ID).
		WithField("workflowRunID", info.WorkflowExecution.RunID).
		WithField("backupID", input.RestoreBackupParams.BackupID).
		Debug("restoring backup")

	svc := ark.NewARKService(org, cluster, a.db, logrusLogger)
	backupsSvc := svc.GetBackupsService()

	backup, err := backupsSvc.GetModelByID(input.RestoreBackupParams.BackupID)
	if err != nil {
		return err
	}

	err = svc.GetDeploymentsService().Deploy(a.helmService, &backup.Bucket, true,
		input.RestoreBackupParams.UseClusterSecret, input.RestoreBackupParams.ServiceAccountRoleARN,
		input.RestoreBackupParams.UseProviderSecret)
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
		err = WaitingForRestoreToFinish(restoresSvc, sync.NewRestoresSyncService(org, a.db, logrusLogger),
			cluster, restore, logrusLogger, a.disasterRecoveryConfig.Ark.RestoreWaitTimeout)
	}
	if err != nil {
		return err
	}

	err = svc.GetDeploymentsService().Remove(a.helmService)
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
		} else if r.Status == "PartiallyFailed" || r.Status == "Failed" {
			return cadence.NewCustomError(ErrReasonRestoreFailed, fmt.Sprintf("backup restore status: %s", r.Status))
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
