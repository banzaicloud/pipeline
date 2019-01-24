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
	"github.com/banzaicloud/pipeline/model"
	"github.com/goph/emperror"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
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
	var existingCluster ClusterModel

	err := c.db.First(&existingCluster, map[string]interface{}{"name": name, "organization_id": organizationID}).Error
	if gorm.IsRecordNotFoundError(err) {
		return false, nil
	} else if err != nil {
		return false, errors.Wrap(err, "could not check cluster existence")
	}

	return existingCluster.ID != 0, nil
}

// All returns all cluster instances for an organization.
func (c *Clusters) All() ([]*model.ClusterModel, error) {
	var clusters []*model.ClusterModel

	err := c.db.Find(&clusters).Error
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch clusters")
	}

	return clusters, nil
}

// FindByOrganization returns all cluster instances for an organization.
func (c *Clusters) FindByOrganization(organizationID uint) ([]*model.ClusterModel, error) {
	var clusters []*model.ClusterModel

	err := c.db.Find(&clusters, map[string]interface{}{"organization_id": organizationID}).Error
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch clusters")
	}

	return clusters, nil
}

// FindOneByID returns a cluster instance for an organization by cluster ID.
func (c *Clusters) FindOneByID(organizationID uint, clusterID uint) (*model.ClusterModel, error) {
	return c.findOneBy(organizationID, "id", clusterID)
}

// FindOneByName returns a cluster instance for an organization by cluster name.
func (c *Clusters) FindOneByName(organizationID uint, clusterName string) (*model.ClusterModel, error) {
	return c.findOneBy(organizationID, "name", clusterName)
}

type clusterModelNotFoundError struct {
	cluster        interface{}
	organizationID uint
}

func (e *clusterModelNotFoundError) Error() string {
	return "cluster not found"
}

func (e *clusterModelNotFoundError) Context() []interface{} {
	return []interface{}{
		"cluster", e.cluster,
		"organization", e.organizationID,
	}
}

func (e *clusterModelNotFoundError) NotFound() bool {
	return true
}

// findOneBy returns a cluster instance for an organization by cluster name.
func (c *Clusters) findOneBy(organizationID uint, field string, criteria interface{}) (*model.ClusterModel, error) {
	cluster := model.ClusterModel{
		OrganizationId: organizationID,
	}

	switch field {
	case "id":
		id, ok := criteria.(uint)
		if !ok {
			return nil, errors.New("criteria is not a valid uint value for id")
		}

		cluster.ID = id

	case "name":
		name, ok := criteria.(string)
		if !ok {
			return nil, errors.New("criteria is not a valid string value for name")
		}

		cluster.Name = name
	}

	err := c.db.Where(cluster).Preload("ScaleOptions").First(&cluster).Error
	if gorm.IsRecordNotFoundError(err) {
		return nil, errors.WithStack(&clusterModelNotFoundError{
			cluster:        criteria,
			organizationID: organizationID,
		})
	} else if err != nil {
		return nil, emperror.With(
			errors.Wrapf(err, "could not get cluster by %s", field),
			"cluster", criteria,
			"organization", organizationID,
		)
	}

	return &cluster, nil
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
		return nil, errors.Wrap(err, "could not fetch clusters")
	}

	return clusters, nil
}
