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

package clusterfeature

import (
	"encoding/json"

	"github.com/banzaicloud/pipeline/cluster"
	"github.com/goph/emperror"
	"github.com/jinzhu/gorm"
	"golang.org/x/net/context"
)

// ClusterRepository collects persistence related operations
type ClusterRepository interface {
	// IsClusterReady checks whether the cluster is ready for features (eg.: exists and it's running)
	IsClusterReady(ctx context.Context, clusterId uint) (bool, error)

	// GetCluster retrieves the cluster representation based on the cluster identifier
	GetCluster(ctx context.Context, clusterId uint) (cluster.CommonCluster, error)
}

// clusterGetter restricts the external dependencies for the repository
type clusterGetter interface {
	GetClusterByIDOnly(ctx context.Context, clusterID uint) (cluster.CommonCluster, error)
}

//
type featureClusterRepository struct {
	clusterGetter clusterGetter
}

func (fcs *featureClusterRepository) GetCluster(ctx context.Context, clusterId uint) (cluster.CommonCluster, error) {

	cluster, err := fcs.clusterGetter.GetClusterByIDOnly(ctx, clusterId)
	if err != nil {
		return nil, emperror.WrapWith(err, "failed to retrieve cluster", "clusterid", clusterId)
	}

	return cluster, nil
}

func (fcs *featureClusterRepository) IsClusterReady(ctx context.Context, clusterId uint) (bool, error) {
	cluster, err := fcs.GetCluster(ctx, clusterId)
	if err != nil {
		return false, err
	}

	isReady, err := cluster.IsReady()
	if err != nil {
		return false, emperror.WrapWith(err, "failed to check cluster", "clusterid", clusterId)
	}

	return isReady, err
}

func NewClusterRepository(getter clusterGetter) ClusterRepository {
	return &featureClusterRepository{
		clusterGetter: getter,
	}
}

// FeatureRepository collects persistence related operations
type FeatureRepository interface {
	SaveFeature(ctx context.Context, clusterId uint, feature Feature) (uint, error)
	GetFeature(ctx context.Context, clusterId uint, feature Feature) (*Feature, error)
	UpdateFeatureStatus(ctx context.Context, clusterId uint, feature Feature, status string) (*Feature, error)
}

// featureRepository component in charge for executing persistence operation on Features
type featureRepository struct {
	db *gorm.DB
}

func (fr *featureRepository) SaveFeature(ctx context.Context, clusterId uint, feature Feature) (uint, error) {

	// encode the spec
	featureSpec, err := json.Marshal(feature.Spec)
	if err != nil {
		return 0, emperror.WrapWith(err, "failed to marshal feature spec", "feature", feature.Name)
	}

	cfModel := ClusterFeatureModel{
		Name:      feature.Name,
		Spec:      featureSpec,
		ClusterID: clusterId,
		Status:    STATUS_PENDING,
	}

	err = fr.db.Save(&cfModel).Error
	if err != nil {
		if err != nil {
			return 0, emperror.WrapWith(err, "failed to persist feature", "feature", feature.Name)
		}
	}

	return cfModel.ID, nil
}

func (fr *featureRepository) GetFeature(ctx context.Context, clusterId uint, feature Feature) (*Feature, error) {
	fm := ClusterFeatureModel{}

	err := fr.db.First(&fm, map[string]interface{}{"Name": feature.Name, "cluster_id": clusterId}).Error

	if gorm.IsRecordNotFoundError(err) {
		return nil, emperror.WrapWith(err, "cluster feature not found", "feature-name", feature.Name)
	} else if err != nil {
		return nil, emperror.Wrap(err, "could not retrieve feature")
	}

	return fr.modelToFeature(&fm)
}

func (fr *featureRepository) UpdateFeatureStatus(ctx context.Context, clusterId uint, feature Feature, status string) (*Feature, error) {

	fm := ClusterFeatureModel{
		ClusterID: clusterId,
		Name:      feature.Name,
	}

	err := fr.db.Model(&fm).Update("status", status).Error
	if err != nil {
		return nil, emperror.Wrap(err, "could not update feature status")
	}

	return fr.modelToFeature(&fm)
}

// NewClusters returns a new Clusters instance.
func NewFeatureRepository(db *gorm.DB) FeatureRepository {
	return &featureRepository{db: db}
}

func (fr *featureRepository) modelToFeature(cfm *ClusterFeatureModel) (*Feature, error) {
	f := Feature{
		Name:   cfm.Name,
		Status: cfm.Status,
	}

	if err := json.Unmarshal(cfm.Spec, &f.Spec); err != nil {
		return nil, emperror.Wrap(err, "failed to retrieve (unmarsha) feature spec")
	}

	return &f, nil
}
