// Copyright © 2019 Banzai Cloud
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

package clusteradapter

import (
	"context"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter/clustermodel"
	"github.com/banzaicloud/pipeline/pkg/sdk/brn"
	"github.com/banzaicloud/pipeline/src/model"
)

// Store is a Cluster persistence implementation.
type Store struct {
	db       *gorm.DB
	clusters *Clusters
}

// NewStore returns a new Store.
func NewStore(db *gorm.DB, clusters *Clusters) Store {
	return Store{
		db:       db,
		clusters: clusters,
	}
}

// GetCluster returns a generic Cluster.
// Returns a NotFoundError when the cluster cannot be found.
func (s Store) GetCluster(ctx context.Context, id uint) (cluster.Cluster, error) {
	clusterModel, err := s.findModel(ctx, id)
	if err != nil {
		return cluster.Cluster{}, err
	}

	return clusterModelToEntity(clusterModel), nil
}

func clusterModelToEntity(m *model.ClusterModel) cluster.Cluster {
	return cluster.Cluster{
		ID:             m.ID,
		UID:            m.UID,
		Name:           m.Name,
		OrganizationID: m.OrganizationId,
		Status:         m.Status,
		StatusMessage:  m.StatusMessage,
		Cloud:          m.Cloud,
		Distribution:   m.Distribution,
		Location:       m.Location,
		SecretID:       brn.New(m.OrganizationId, brn.SecretResourceType, m.SecretId),
		ConfigSecretID: brn.New(m.OrganizationId, brn.SecretResourceType, m.ConfigSecretId),
		Tags:           m.Tags,
	}
}

func (s Store) GetClusterByName(ctx context.Context, orgID uint, clusterName string) (cluster.Cluster, error) {
	clusterModel, err := s.findModelByName(ctx, orgID, clusterName)
	if err != nil {
		return cluster.Cluster{}, err
	}

	return clusterModelToEntity(clusterModel), nil
}

// Exists returns true if the cluster exists in the store and is not deleted
func (s Store) Exists(ctx context.Context, id uint) (bool, error) {
	m, err := s.clusters.FindOneByID(0, id)

	if IsClusterNotFoundError(err) {
		return true, nil
	}

	if err != nil {
		return false, err
	}

	if m == nil {
		return false, nil
	}

	return m.DeletedAt == nil, nil
}

func (s Store) findModel(ctx context.Context, id uint) (*model.ClusterModel, error) {
	clusterModel, err := s.clusters.FindOneByID(0, id)
	if err != nil {
		if IsClusterNotFoundError(err) {
			return nil, errors.WithStack(cluster.NotFoundError{ClusterID: id})
		}

		return nil, errors.WrapWithDetails(err, "failed to get cluster", "clusterId", id)
	}

	return clusterModel, nil
}

func (s Store) findModelByName(ctx context.Context, orgID uint, clusterName string) (*model.ClusterModel, error) {
	clusterModel, err := s.clusters.FindOneByName(orgID, clusterName)
	if err != nil {
		if IsClusterNotFoundError(err) {
			return nil, errors.WithStack(cluster.NotFoundError{OrganizationID: orgID, ClusterName: clusterName})
		}

		return nil, errors.WrapWithDetails(err, "failed to get cluster", "clusterName", clusterName, "orgId", orgID)
	}

	return clusterModel, nil
}

// SetStatus sets the cluster status.
func (s Store) SetStatus(ctx context.Context, id uint, status string, message string) error {
	clusterModel, err := s.findModel(ctx, id)
	if err != nil {
		return err
	}

	if status != clusterModel.Status || message != clusterModel.StatusMessage {
		fields := map[string]interface{}{
			"status":        status,
			"statusMessage": message,
		}

		statusHistory := clustermodel.StatusHistoryModel{
			ClusterID:   clusterModel.ID,
			ClusterName: clusterModel.Name,

			FromStatus:        clusterModel.Status,
			FromStatusMessage: clusterModel.StatusMessage,
			ToStatus:          status,
			ToStatusMessage:   message,
		}
		if err := s.db.Save(&statusHistory).Error; err != nil {
			return errors.WrapIfWithDetails(err, "failed to save status history", "cluster_id", id)
		}

		err := s.db.Model(&clusterModel).Updates(fields).Error
		if err != nil {
			return errors.WrapIfWithDetails(err, "failed to update cluster status", "cluster_id", id)
		}
	}

	return nil
}
