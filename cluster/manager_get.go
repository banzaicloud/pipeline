package cluster

import (
	"context"

	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// GetClusters returns the cluster instances for an organization ID.
func (m *Manager) GetClusters(ctx context.Context, organizationID uint) ([]CommonCluster, error) {
	logger := m.getLogger(ctx).WithFields(logrus.Fields{
		"organization": organizationID,
	})

	logger.Debug("fetching clusters from database")

	clusterModels, err := m.clusters.FindByOrganization(organizationID)
	if err != nil {
		return nil, err
	}

	var clusters []CommonCluster

	for _, clusterModel := range clusterModels {
		logger := logger.WithField("cluster", clusterModel.Name)
		logger.Debug("converting cluster model to common cluster")

		cluster, err := GetCommonClusterFromModel(clusterModel)
		if err != nil {
			logger.Error("converting cluster model to common cluster failed")

			continue
		}

		clusters = append(clusters, cluster)
	}

	return clusters, nil
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
