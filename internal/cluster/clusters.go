package cluster

import (
	"github.com/banzaicloud/pipeline/model"
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

	return existingCluster.ID == 0, nil
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
