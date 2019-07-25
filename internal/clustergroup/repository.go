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

package clustergroup

import (
	"emperror.dev/errors"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/clustergroup/api"
)

// ClusterGroupRepository
type ClusterGroupRepository struct {
	db     *gorm.DB
	logger logrus.FieldLogger
}

// NewClusterGroupRepository returns a new ClusterGroupRepository instance.
func NewClusterGroupRepository(
	db *gorm.DB,
	logger logrus.FieldLogger,
) *ClusterGroupRepository {
	return &ClusterGroupRepository{
		db:     db,
		logger: logger,
	}
}

// FindOne returns a cluster group instance for an organization by clusterGroupID.
func (g *ClusterGroupRepository) FindOne(cg ClusterGroupModel) (*ClusterGroupModel, error) {
	if cg.ID == 0 && len(cg.Name) == 0 {
		return nil, &invalidClusterGroupCreateRequestError{
			message: "either clusterGroupID or name is required",
		}
	}
	var result ClusterGroupModel
	err := g.db.Where(cg).Preload("Members").Preload("FeatureParams").First(&result).Error
	if gorm.IsRecordNotFoundError(err) {
		return nil, errors.WithStack(&clusterGroupNotFoundError{
			clusterGroup: cg,
		})
	}
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "could not find cluster group", "ID", cg.ID)
	}

	return &result, nil
}

// FindAll returns all cluster groups
func (g *ClusterGroupRepository) FindAll(orgID uint) ([]*ClusterGroupModel, error) {
	var cgroups []*ClusterGroupModel

	err := g.db.Where(ClusterGroupModel{
		OrganizationID: orgID,
	}).Preload("Members").Preload("FeatureParams").Find(&cgroups).Error
	if err != nil {
		return nil, errors.WrapIf(err, "could not find cluster groups")
	}

	return cgroups, nil
}

// Create persists a cluster group
func (g *ClusterGroupRepository) Create(name string, orgID uint, memberClusterModels []MemberClusterModel) (*uint, error) {
	clusterGroupModel := &ClusterGroupModel{
		Name:           name,
		OrganizationID: orgID,
		Members:        memberClusterModels,
	}

	err := g.db.Save(clusterGroupModel).Error
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "error creating cluster group", "name", name)
	}
	return &clusterGroupModel.ID, nil
}

// UpdateMembers updates cluster group members
func (g *ClusterGroupRepository) UpdateMembers(cgroup *api.ClusterGroup, newMembers map[uint]api.Cluster) error {
	cgModel, err := g.FindOne(ClusterGroupModel{
		ID: cgroup.Id,
	})
	if err != nil {
		return err
	}
	updatedMembers := make([]MemberClusterModel, 0)

	for _, member := range cgModel.Members {
		// delete members
		if _, ok := newMembers[member.ClusterID]; !ok {
			err = g.db.Delete(member).Error
			if err != nil {
				return errors.WrapIfWithDetails(err, "could not delete member cluster", "clusterGroupID", cgroup.Id, "clusterID", member.ClusterID)
			}
		} else {
			updatedMembers = append(updatedMembers, member)
		}
		delete(newMembers, member.ClusterID)
	}

	// add new ones
	for _, member := range newMembers {
		updatedMembers = append(updatedMembers, MemberClusterModel{
			ClusterID: member.GetID(),
		})
	}

	cgModel.Members = updatedMembers
	err = g.db.Save(cgModel).Error
	if err != nil {
		return errors.WrapIfWithDetails(err, "could not update member clusters", "clusterGroupID", cgroup.Id)
	}
	return nil
}

// Delete deletes a cluster group
func (g *ClusterGroupRepository) Delete(cgroup *ClusterGroupModel) error {
	for _, fp := range cgroup.FeatureParams {
		err := g.db.Delete(fp).Error
		if err != nil {
			return err
		}
	}

	for _, member := range cgroup.Members {
		err := g.db.Delete(member).Error
		if err != nil {
			return errors.WrapIfWithDetails(err, "could not delete member cluster", "clusterGroupID", cgroup.ID, "clusterID", member.ClusterID)
		}
	}

	err := g.db.Delete(cgroup).Error
	if err != nil {
		return errors.WrapIfWithDetails(err, "could not delete cluster group", "clusterGroupID", cgroup.ID)
	}

	return nil
}

// GetFeature gets a feature for a cluster
func (g *ClusterGroupRepository) GetFeature(clusterGroupID uint, featureName string) (*ClusterGroupFeatureModel, error) {
	var result ClusterGroupFeatureModel
	err := g.db.Where(ClusterGroupFeatureModel{
		ClusterGroupID: clusterGroupID,
		Name:           featureName,
	}).First(&result).Error

	if gorm.IsRecordNotFoundError(err) {
		return nil, errors.WithStack(&featureRecordNotFoundError{})
	}

	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "could not find cluster group feature", "clusterGroupID", clusterGroupID, "name", featureName)
	}

	return &result, nil
}

// SaveFeature persists a cluster group feature
func (g *ClusterGroupRepository) SaveFeature(feature *ClusterGroupFeatureModel) error {
	if len(feature.Properties) == 0 {
		feature.Properties = []byte("{}")
	}
	err := g.db.Save(feature).Error
	if err != nil {
		return errors.WrapIfWithDetails(err, "error saving cluster group feature", "name", feature.Name, "clusterGroupID", feature.ClusterGroupID)
	}
	return nil
}

// GetAllFeatures gets all features for a cluster group
func (g *ClusterGroupRepository) GetAllFeatures(clusterGroupID uint) ([]ClusterGroupFeatureModel, error) {
	var results []ClusterGroupFeatureModel
	err := g.db.Find(&results, ClusterGroupFeatureModel{
		ClusterGroupID: clusterGroupID,
	}).Error

	if gorm.IsRecordNotFoundError(err) {
		return nil, errors.WithStack(&recordNotFoundError{})
	}

	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "could not find cluster group features", "clusterGroupID", clusterGroupID)
	}

	return results, nil
}

// FindMemberClusterByID returns a MemberClusterModel for a cluster ID
func (g *ClusterGroupRepository) FindMemberClusterByID(clusterID uint) (*MemberClusterModel, error) {
	var result MemberClusterModel
	err := g.db.Where(MemberClusterModel{
		ClusterID: clusterID,
	}).First(&result).Error
	if gorm.IsRecordNotFoundError(err) {
		return nil, errors.WithStack(&recordNotFoundError{})
	}

	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "could not find member cluster", "clusterID", clusterID)
	}

	return &result, nil
}
