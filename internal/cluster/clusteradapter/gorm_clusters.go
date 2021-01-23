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
	"emperror.dev/errors"
	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter/clustermodel"
	"github.com/banzaicloud/pipeline/src/model"
)

// Clusters acts as a repository interface for clusters.
type Clusters struct {
	db *gorm.DB
}

// NewClusters returns a new Clusters instance.
func NewClusters(db *gorm.DB) *Clusters {
	return &Clusters{db: db}
}

// Exists checks if a given cluster exists within an organization.
func (c *Clusters) Exists(organizationID uint, name string) (bool, error) {
	var existingCluster clustermodel.ClusterModel

	err := c.db.First(&existingCluster, map[string]interface{}{"name": name, "organization_id": organizationID}).Error
	if gorm.IsRecordNotFoundError(err) {
		return false, nil
	} else if err != nil {
		return false, errors.WrapIf(err, "could not check cluster existence")
	}

	return existingCluster.ID != 0, nil
}

// All returns all cluster instances for an organization.
func (c *Clusters) All() ([]*model.ClusterModel, error) {
	var clusters []*model.ClusterModel

	err := c.db.Find(&clusters).Error
	if err != nil {
		return nil, errors.WrapIf(err, "could not fetch clusters")
	}

	return clusters, nil
}

// FindByOrganization returns all cluster instances for an organization.
func (c *Clusters) FindByOrganization(organizationID uint) ([]*model.ClusterModel, error) {
	var clusters []*model.ClusterModel

	err := c.db.Find(&clusters, map[string]interface{}{"organization_id": organizationID}).Error
	if err != nil {
		return nil, errors.WrapIf(err, "could not fetch clusters")
	}

	return clusters, nil
}

// FindOneByID returns a cluster instance for an organization by cluster ID.
func (c *Clusters) FindOneByID(organizationID uint, clusterID uint) (*model.ClusterModel, error) {
	cluster, err := c.findOneBy(model.ClusterModel{
		OrganizationId: organizationID,
		ID:             clusterID,
	})
	if err != nil {
		return nil, errors.WrapIf(err, "could not find cluster by ID")
	}
	return cluster, nil
}

// FindOneByName returns a cluster instance for an organization by cluster name.
func (c *Clusters) FindOneByName(organizationID uint, clusterName string) (*model.ClusterModel, error) {
	cluster, err := c.findOneBy(model.ClusterModel{
		OrganizationId: organizationID,
		Name:           clusterName,
	})
	if err != nil {
		return nil, errors.WrapIf(err, "could not find cluster by name")
	}
	return cluster, nil
}

type clusterModelNotFoundError struct {
	cluster model.ClusterModel
}

func (e *clusterModelNotFoundError) Error() string {
	return "cluster not found"
}

func (e *clusterModelNotFoundError) Context() []interface{} {
	return []interface{}{
		"clusterID", e.cluster.ID,
		"clusterName", e.cluster.Name,
		"organizationID", e.cluster.OrganizationId,
	}
}

func (e *clusterModelNotFoundError) NotFound() bool {
	return true
}

// IsClusterNotFoundError returns true if the passed in error designates a cluster not found error
func IsClusterNotFoundError(err error) bool {
	var notFoundErr interface {
		NotFound() bool
	}

	return errors.As(err, &notFoundErr) && notFoundErr.NotFound()
}

// findOneBy returns a cluster instance for an organization by cluster name.
func (c *Clusters) findOneBy(cluster model.ClusterModel) (*model.ClusterModel, error) {
	if cluster.ID == 0 && cluster.Name == "" {
		return nil, errors.New("no cluster identifying field specified")
	}
	var result model.ClusterModel
	err := c.db.Where(cluster).Preload("ScaleOptions").First(&result).Error
	if gorm.IsRecordNotFoundError(err) {
		return nil, errors.WithStack(&clusterModelNotFoundError{
			cluster: cluster,
		})
	}
	if err != nil {
		return nil, errors.WithDetails(err,
			"clusterID", cluster.ID,
			"clusterName", cluster.Name,
			"organizationID", cluster.OrganizationId,
		)
	}

	return &result, nil
}

// FindBySecret returns all cluster instances for an organization filtered by secret.
func (c *Clusters) FindBySecret(organizationID uint, secretID string) ([]*model.ClusterModel, error) {
	var clusters []*model.ClusterModel

	err := c.db.Find(
		&clusters,
		map[string]interface{}{
			"organization_id": organizationID,
			"secret_id":       secretID,
		},
	).Error
	if err != nil {
		return nil, errors.WrapIf(err, "could not fetch clusters")
	}

	return clusters, nil
}

// GetConfigSecretIDByClusterID returns the kubeconfig's secretID stored in DB
func (c *Clusters) GetConfigSecretIDByClusterID(organizationID uint, clusterID uint) (string, error) {
	cluster := model.ClusterModel{ID: clusterID}

	if err := c.db.Where(cluster).Select("config_secret_id").First(&cluster).Error; err != nil {
		return "", errors.WrapIf(err, "could not get ConfigSecretID")
	}

	return cluster.ConfigSecretId, nil
}

// FindNextWithGreaterID returns the next cluster <orgID, clusterID> tuple that is greater than the passed in clusterID
func (c *Clusters) FindNextWithGreaterID(clusterID uint) (uint, uint, error) {
	cluster := model.ClusterModel{}

	err := c.db.Where("id > ?", clusterID).First(&cluster).Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return 0, 0, errors.WithStack(&clusterModelNotFoundError{
				cluster: cluster,
			})
		}
		return 0, 0, err
	}

	return cluster.OrganizationId, cluster.ID, nil
}
