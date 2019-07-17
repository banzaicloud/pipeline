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

	"emperror.dev/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/model"
)

// GetClusters returns the cluster instances for an organization ID.
func (m *Manager) GetClusters(ctx context.Context, organizationID uint) ([]CommonCluster, error) {
	logger := m.getLogger(ctx).WithFields(logrus.Fields{
		"organization": organizationID,
	})

	clusterModels, err := m.clusters.FindByOrganization(organizationID)
	if err != nil {
		return nil, err
	}

	var clusters []CommonCluster

	for _, clusterModel := range clusterModels {
		logger := logger.WithField("cluster", clusterModel.Name)

		cluster, err := GetCommonClusterFromModel(clusterModel)
		if err != nil {
			logger.Errorf("converting cluster model to common cluster failed: %s", err.Error())

			continue
		}

		clusters = append(clusters, cluster)
	}

	return clusters, nil
}

// GetAllClusters returns all cluster instances.
func (m *Manager) GetAllClusters(ctx context.Context) ([]CommonCluster, error) {
	logger := m.getLogger(ctx)

	clusterModels, err := m.clusters.All()
	if err != nil {
		return nil, err
	}

	return m.getClustersFromModels(clusterModels, logger), nil
}

// GetClusterByID returns the cluster instance for an organization ID by cluster ID.
func (m *Manager) GetClusterByID(ctx context.Context, organizationID uint, clusterID uint) (CommonCluster, error) {
	logger := m.getLogger(ctx).WithFields(logrus.Fields{
		"organization": organizationID,
		"cluster":      clusterID,
	})

	logger.Debug("getting cluster from database")

	clusterModel, err := m.clusters.FindOneByID(organizationID, clusterID)
	if err != nil {
		return nil, errors.Wrap(err, "could not get cluster from database")
	}

	cluster, err := GetCommonClusterFromModel(clusterModel)
	if err != nil {
		return nil, emperror.Wrap(err, "could not get cluster from model")
	}

	return cluster, nil
}

// GetClusterByIDOnly returns the cluster instance by cluster ID.
func (m *Manager) GetClusterByIDOnly(ctx context.Context, clusterID uint) (CommonCluster, error) {
	logger := m.getLogger(ctx).WithFields(logrus.Fields{
		"cluster": clusterID,
	})

	logger.Debug("getting cluster from database")

	clusterModel, err := m.clusters.FindOneByID(0, clusterID)
	if err != nil {
		return nil, errors.Wrap(err, "could not get cluster from database")
	}

	cluster, err := GetCommonClusterFromModel(clusterModel)
	if err != nil {
		return nil, emperror.Wrap(err, "could not get cluster from model")
	}

	return cluster, nil
}

// GetClusterByName returns the cluster instance for an organization ID by cluster name.
func (m *Manager) GetClusterByName(ctx context.Context, organizationID uint, clusterName string) (CommonCluster, error) {
	logger := m.getLogger(ctx).WithFields(logrus.Fields{
		"organization": organizationID,
		"cluster":      clusterName,
	})

	logger.Debug("getting cluster from database")

	clusterModel, err := m.clusters.FindOneByName(organizationID, clusterName)
	if err != nil {
		return nil, errors.Wrap(err, "could not get cluster from database")
	}

	cluster, err := GetCommonClusterFromModel(clusterModel)
	if err != nil {
		return nil, emperror.Wrap(err, "could not get cluster from model")
	}

	return cluster, nil
}

// GetClustersBySecretID returns the cluster instance for an organization ID by secret ID.
func (m *Manager) GetClustersBySecretID(ctx context.Context, organizationID uint, secretID string) ([]CommonCluster, error) {
	logger := m.getLogger(ctx).WithFields(logrus.Fields{
		"organization": organizationID,
		"secret":       secretID,
	})

	logger.Debug("getting cluster from database")

	clusterModels, err := m.clusters.FindBySecret(organizationID, secretID)
	if err != nil {
		return nil, errors.Wrap(err, "could not get cluster from database")
	}

	return m.getClustersFromModels(clusterModels, logger), nil
}

func (m *Manager) getClusterFromModel(clusterModel *model.ClusterModel) (CommonCluster, error) {
	return GetCommonClusterFromModel(clusterModel)
}

func (m *Manager) getClustersFromModels(clusterModels []*model.ClusterModel, logger logrus.FieldLogger) []CommonCluster {
	var clusters []CommonCluster

	for _, clusterModel := range clusterModels {
		logger := logger.WithField("cluster", clusterModel.Name)

		cluster, err := m.getClusterFromModel(clusterModel)
		if err != nil {
			logger.Errorf("converting cluster model to common cluster failed: %s", err.Error())

			continue
		}

		clusters = append(clusters, cluster)
	}

	return clusters
}
