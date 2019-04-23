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
	"bytes"
	"context"
	"encoding/json"

	"github.com/goph/emperror"
	arkAPI "github.com/heptio/ark/pkg/apis/ark/v1"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/ark"
	"github.com/banzaicloud/pipeline/internal/ark/api"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
)

// RestoresSyncService is for syncing restores from ARK
type RestoresSyncService struct {
	org    *auth.Organization
	db     *gorm.DB
	logger logrus.FieldLogger
}

// NewRestoresSyncService returns an initialized RestoresSyncService
func NewRestoresSyncService(
	org *auth.Organization,
	db *gorm.DB,
	logger logrus.FieldLogger,
) *RestoresSyncService {

	return &RestoresSyncService{
		org:    org,
		db:     db,
		logger: logger,
	}
}

// SyncRestores syncs restores from ARK for every cluster within the organization
func (s *RestoresSyncService) SyncRestores(clusterManager api.ClusterManager) error {

	clusters, err := clusterManager.GetClusters(context.Background(), s.org.ID)
	if err != nil {
		return err
	}

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

		err = s.SyncRestoresForCluster(cluster)
		if err != nil && errors.Cause(err) != gorm.ErrRecordNotFound {
			log.Error(err)
		}
	}

	return nil
}

func (s *RestoresSyncService) SyncRestoresForCluster(cluster api.Cluster) error {

	deployments := ark.DeploymentsServiceFactory(s.org, cluster, s.db, s.logger)

	deployment, err := deployments.GetActiveDeployment()
	if err != nil {
		return err
	}

	restoresSvc := ark.RestoresServiceFactory(s.org, deployments, s.db, s.logger)

	restores, err := restoresSvc.ListFromARK()
	if err != nil {
		return err
	}

	for _, restore := range restores {
		err = s.syncRestore(restoresSvc, cluster, deployment, restore)
		if err != nil {
			return emperror.Wrap(err, "error persisting restore")
		}
	}

	return nil
}

func (s *RestoresSyncService) syncRestore(
	svc *ark.RestoresService,
	cluster api.Cluster,
	deployment *ark.ClusterBackupDeploymentsModel,
	restore arkAPI.Restore,
) error {

	log := s.logger.WithField("restore-name", restore.Name)
	log.Debugf("syncing...")

	req := &api.PersistRestoreRequest{
		BucketID:  deployment.BucketID,
		ClusterID: cluster.GetID(),
		Restore:   &restore,
	}

	// get results for completed restores
	if restore.Status.Phase == arkAPI.RestorePhaseCompleted {
		result, err := s.getRestoreResultFromObjectStore(req)
		if err != nil {
			s.logger.Error(err)
		} else {
			req.Results = result
		}
	}

	_, err := svc.Persist(req)
	if err != nil {
		return err
	}

	log.Debugf("synced")

	return nil
}

func (s *RestoresSyncService) getRestoreResultFromObjectStore(req *api.PersistRestoreRequest) (
	*api.RestoreResults, error) {

	bs := ark.BucketsServiceFactory(s.org, s.db, s.logger)
	bucket, err := bs.GetByID(req.BucketID)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	err = bs.StreamRestoreResultsFromObjectStore(bucket, req.Restore.Spec.BackupName, req.Restore.Name, buf)
	if err != nil {
		return nil, err
	}

	var result api.RestoreResults
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
