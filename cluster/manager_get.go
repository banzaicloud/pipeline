package cluster

import (
	"context"

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
