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

package clusterfeatureadapter

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"
	"github.com/spf13/cast"
	"logur.dev/logur"

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/common"
)

// TableName constants
const (
	clusterFeatureTableName = "cluster_features"
)

type featureSpec map[string]interface{}

func (fs *featureSpec) Scan(src interface{}) error {
	value, err := cast.ToStringE(src)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(value), fs)
}

func (fs featureSpec) Value() (driver.Value, error) {
	v, err := json.Marshal(fs)
	if err != nil {
		return "", err
	}
	return v, nil
}

// clusterFeatureModel describes the cluster group model.
type clusterFeatureModel struct {
	// injecting timestamp fields
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Name      string `gorm:"unique_index:idx_cluster_feature_cluster_id_name"`
	Status    string
	ClusterId uint        `gorm:"unique_index:idx_cluster_feature_cluster_id_name"`
	Spec      featureSpec `gorm:"type:text"`
	CreatedBy uint
}

// TableName changes the default table name.
func (cfm clusterFeatureModel) TableName() string {
	return clusterFeatureTableName
}

// String method prints formatted cluster fields.
func (cfm clusterFeatureModel) String() string {
	return fmt.Sprintf("Id: %d, Creation date: %s, Name: %s", cfm.ID, cfm.CreatedAt, cfm.Name)
}

// GORMFeatureRepository implements feature persistence in RDBMS using GORM.
// TODO: write integration tests
type GORMFeatureRepository struct {
	db     *gorm.DB
	logger common.Logger
}

// NewGormFeatureRepository returns a feature repository persisting feature state into database using GORM.
func NewGormFeatureRepository(db *gorm.DB, logger common.Logger) GORMFeatureRepository {
	return GORMFeatureRepository{
		db:     db,
		logger: logger,
	}
}

// GetFeatures returns features stored in the repository for the specified cluster.
func (r GORMFeatureRepository) GetFeatures(ctx context.Context, clusterID uint) ([]clusterfeature.Feature, error) {
	logger := logur.WithFields(r.logger, map[string]interface{}{"clusterID": clusterID})
	logger.Info("retrieving features for cluster")

	var (
		featureModels []clusterFeatureModel
		featureList   []clusterfeature.Feature
	)

	if err := r.db.Find(&featureModels, clusterFeatureModel{ClusterId: clusterID}).Error; err != nil {
		logger.Debug("could not retrieve features")

		return nil, errors.WrapIfWithDetails(err, "could not retrieve features", "clusterID", clusterID)
	}

	// model  --> domain
	for _, feature := range featureModels {
		f, e := r.modelToFeature(feature)
		if e != nil {
			logger.Debug("failed to convert model to feature")
			continue
		}

		featureList = append(featureList, f)
	}

	logger.Info("features list for cluster retrieved")

	return featureList, nil
}

// SaveFeature persists a feature with the specified properties in the database.
func (r GORMFeatureRepository) SaveFeature(ctx context.Context, clusterID uint, featureName string, spec clusterfeature.FeatureSpec, status string) error {
	model := clusterFeatureModel{
		ClusterId: clusterID,
		Name:      featureName,
	}

	if err := r.db.Where(&model).First(&model).Error; err != nil && !gorm.IsRecordNotFoundError(err) {
		return errors.WrapIfWithDetails(err, "failed to query feature", "clusterId", clusterID, "feature", featureName)
	}
	model.Spec = spec
	model.Status = status
	return errors.WrapIfWithDetails(r.db.Save(&model).Error, "failed to save feature", "clusterId", clusterID, "feature", featureName)
}

// GetFeature retrieves a feature by the cluster ID and feature name.
// It returns a "feature not found" error if the feature is not in the database.
func (r GORMFeatureRepository) GetFeature(ctx context.Context, clusterID uint, featureName string) (clusterfeature.Feature, error) {
	fm := clusterFeatureModel{}

	err := r.db.First(&fm, map[string]interface{}{"Name": featureName, "cluster_id": clusterID}).Error

	if gorm.IsRecordNotFoundError(err) {
		return clusterfeature.Feature{}, featureNotFoundError{
			ClusterID:   clusterID,
			FeatureName: featureName,
		}
	} else if err != nil {
		return clusterfeature.Feature{}, errors.WrapIf(err, "could not retrieve feature")
	}

	return r.modelToFeature(fm)
}

// UpdateFeatureStatus sets the status of the specified feature
func (r GORMFeatureRepository) UpdateFeatureStatus(ctx context.Context, clusterID uint, featureName string, status string) error {
	fm := clusterFeatureModel{
		ClusterId: clusterID,
		Name:      featureName,
	}

	return errors.WrapIf(r.db.Find(&fm, fm).Updates(clusterFeatureModel{Status: status}).Error, "could not update feature status")
}

// UpdateFeatureSpec sets the specification of the specified feature
func (r GORMFeatureRepository) UpdateFeatureSpec(ctx context.Context, clusterID uint, featureName string, spec clusterfeature.FeatureSpec) error {

	fm := clusterFeatureModel{ClusterId: clusterID, Name: featureName}

	return errors.WrapIf(r.db.Find(&fm, fm).Updates(clusterFeatureModel{Spec: spec}).Error, "could not update feature spec")
}

func (r GORMFeatureRepository) modelToFeature(cfm clusterFeatureModel) (clusterfeature.Feature, error) {
	f := clusterfeature.Feature{
		Name:   cfm.Name,
		Status: cfm.Status,
		Spec:   cfm.Spec,
	}

	return f, nil
}

// DeleteFeature permanently deletes the feature record
func (r GORMFeatureRepository) DeleteFeature(ctx context.Context, clusterID uint, featureName string) error {

	fm := clusterFeatureModel{ClusterId: clusterID, Name: featureName}

	if err := r.db.Delete(&fm, fm).Error; err != nil {

		return errors.WrapIf(err, "could not delete status")
	}

	return nil

}

type featureNotFoundError struct {
	ClusterID   uint
	FeatureName string
}

func (e featureNotFoundError) Error() string {
	return fmt.Sprintf("Feature %q not found for cluster %d", e.FeatureName, e.ClusterID)
}

func (e featureNotFoundError) Details() []interface{} {
	return []interface{}{
		"clusterId", e.ClusterID,
		"feature", e.FeatureName,
	}
}

func (featureNotFoundError) FeatureNotFound() bool {
	return true
}
