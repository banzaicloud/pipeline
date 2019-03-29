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

package cluster

import (
	"context"

	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// UpdateContext represents the data necessary to do generic cluster update steps/checks.
type UpdateContext struct {
	OrganizationID uint
	UserID         uint
	ClusterID      uint
}

type clusterUpdater interface {
	// Validate validates the cluster update context.
	Validate(ctx context.Context) error

	// Prepare prepares a cluster to be updated.
	Prepare(ctx context.Context) (CommonCluster, error)

	// Update updates a cluster.
	Update(ctx context.Context) error
}

// UpdateCluster updates a cluster.
func (m *Manager) UpdateCluster(ctx context.Context, updateCtx UpdateContext, updater clusterUpdater) error {
	logger := m.getLogger(ctx).WithFields(logrus.Fields{
		"organization": updateCtx.OrganizationID,
		"user":         updateCtx.UserID,
		"cluster":      updateCtx.ClusterID,
	})

	logger.Debug("validating update context")

	err := updater.Validate(ctx)
	if err != nil {
		return errors.WithMessage(err, "cluster update validation failed")
	}

	logger.Debug("preparing cluster update")

	cluster, err := updater.Prepare(ctx)
	if err != nil {
		return errors.WithMessage(err, "could not prepare cluster")
	}

	timer, err := m.getPrometheusTimer(cluster.GetCloud(), cluster.GetLocation(), pkgCluster.Updating, cluster.GetOrganizationId(), cluster.GetName())
	if err != nil {
		return err
	}

	if err := cluster.SetStatus(pkgCluster.Updating, pkgCluster.UpdatingMessage); err != nil {
		return emperror.With(err, "could not update cluster status")
	}

	logger.Info("updating cluster")
	errorHandler := m.getClusterErrorHandler(ctx, cluster)

	go func() {
		defer emperror.HandleRecover(errorHandler.WithStatus(pkgCluster.Warning, "internal error while updating cluster"))

		err := m.updateCluster(ctx, updateCtx, cluster, updater)
		if err != nil {
			errorHandler.Handle(err)
			return
		}
		timer.ObserveDuration()
	}()

	return nil
}

func (m *Manager) updateCluster(ctx context.Context, updateCtx UpdateContext, cluster CommonCluster, updater clusterUpdater) error {
	logger := m.getLogger(ctx).WithFields(logrus.Fields{
		"organization": updateCtx.OrganizationID,
		"user":         updateCtx.UserID,
		"cluster":      updateCtx.ClusterID,
	})

	logger.Info("updating cluster")

	if err := updater.Update(ctx); err != nil {
		if setErr := cluster.SetStatus(pkgCluster.Warning, err.Error()); setErr != nil {
			log.Error(setErr, "could not set cluster status")
		} else {
			m.events.ClusterUpdated(updateCtx.ClusterID)
		}

		return emperror.Wrap(err, "error updating cluster")
	}

	if err := cluster.SetStatus(pkgCluster.Running, pkgCluster.RunningMessage); err != nil {
		return emperror.Wrap(err, "could not update cluster status")
	}
	m.events.ClusterUpdated(updateCtx.ClusterID)

	logger.Info("cluster updated successfully")

	return nil
}
