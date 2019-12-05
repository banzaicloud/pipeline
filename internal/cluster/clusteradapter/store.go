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
	"github.com/banzaicloud/pipeline/pkg/brn"
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

	return cluster.Cluster{
		ID:             clusterModel.ID,
		UID:            clusterModel.UID,
		Name:           clusterModel.Name,
		OrganizationID: clusterModel.OrganizationId,
		Status:         clusterModel.Status,
		StatusMessage:  clusterModel.StatusMessage,
		Cloud:          clusterModel.Cloud,
		Distribution:   clusterModel.Distribution,
		Location:       clusterModel.Location,
		SecretID:       brn.New(clusterModel.OrganizationId, brn.SecretResourceType, clusterModel.SecretId),
		ConfigSecretID: brn.New(clusterModel.OrganizationId, brn.SecretResourceType, clusterModel.ConfigSecretId),
	}, nil
}

func (s Store) findModel(ctx context.Context, id uint) (*model.ClusterModel, error) {
	clusterModel, err := s.clusters.FindOneByID(0, id)
	if err != nil {
		if IsClusterNotFoundError(err) {
			return nil, errors.WithStack(cluster.NotFoundError{ID: id})
		}

		return nil, errors.WrapWithDetails(err, "failed to get cluster", "cluster_id", id)
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

		statusHistory := StatusHistoryModel{
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
