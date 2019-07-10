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

	"emperror.dev/emperror"
	"github.com/goph/logur"
	"github.com/jinzhu/gorm"
	"github.com/spf13/cast"

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
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

	Name      string
	Status    string
	ClusterId uint
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

// gormFeatureRepository component in charge for executing persistence operation on Features.
// TODO: write integration tests
type gormFeatureRepository struct {
	logger logur.Logger
	db     *gorm.DB
}

// NewGormFeatureRepository returns a feature repository persisting feature state into database using Gorm.
func NewGormFeatureRepository(logger logur.Logger, db *gorm.DB) clusterfeature.FeatureRepository {
	return &gormFeatureRepository{
		logger: logur.WithFields(logger, map[string]interface{}{"gorm-feature-repo": "comp"}),
		db:     db}
}

func (r *gormFeatureRepository) SaveFeature(ctx context.Context, clusterID uint, featureName string, featureSpec clusterfeature.FeatureSpec) (uint, error) {
	cfModel := clusterFeatureModel{
		Name:      featureName,
		Spec:      featureSpec,
		ClusterId: clusterID,
		Status:    string(clusterfeature.FeatureStatusPending),
	}

	err := r.db.Save(&cfModel).Error
	if err != nil {
		return 0, emperror.WrapWith(err, "failed to persist feature", "feature", featureName)
	}

	return cfModel.ID, nil
}

// GetFeature retrieves a featuer by the cluster id and the feature name.
// Returns (nil, nil) in case the feature is not found
func (r *gormFeatureRepository) GetFeature(ctx context.Context, clusterID uint, featureName string) (*clusterfeature.Feature, error) {
	fm := clusterFeatureModel{}

	err := r.db.First(&fm, map[string]interface{}{"Name": featureName, "cluster_id": clusterID}).Error

	if gorm.IsRecordNotFoundError(err) {
		return nil, nil
	} else if err != nil {
		return nil, emperror.Wrap(err, "could not retrieve feature")
	}

	return r.modelToFeature(fm)
}

// UpdateFeatureStatus updates the status of the feature
func (r *gormFeatureRepository) UpdateFeatureStatus(ctx context.Context, clusterID uint, featureName string, status string) (*clusterfeature.Feature, error) {
	fm := clusterFeatureModel{
		ClusterId: clusterID,
		Name:      featureName,
	}

	if err := r.db.Find(&fm, fm).Updates(clusterFeatureModel{Status: status}).Error; err != nil {
		return nil, emperror.Wrap(err, "could not update feature status")
	}

	return r.modelToFeature(fm)
}

// UpdateFeatureStatus updates the status of the feature
func (r *gormFeatureRepository) UpdateFeatureSpec(ctx context.Context, clusterID uint, featureName string, spec clusterfeature.FeatureSpec) (*clusterfeature.Feature, error) {

	fm := clusterFeatureModel{ClusterId: clusterID, Name: featureName}

	if err := r.db.Find(&fm, fm).Updates(clusterFeatureModel{Spec: spec}).Error; err != nil {

		return nil, emperror.Wrap(err, "could not update feature spec")
	}

	return r.modelToFeature(fm)
}

func (r *gormFeatureRepository) modelToFeature(cfm clusterFeatureModel) (*clusterfeature.Feature, error) {
	f := clusterfeature.Feature{
		Name:   cfm.Name,
		Status: cfm.Status,
		Spec:   cfm.Spec,
	}

	return &f, nil
}

// DeleteFeature permanently deletes the feature record
func (r *gormFeatureRepository) DeleteFeature(ctx context.Context, clusterID uint, featureName string) error {

	fm := clusterFeatureModel{ClusterId: clusterID, Name: featureName}

	if err := r.db.Delete(&fm, fm).Error; err != nil {

		return emperror.Wrap(err, "could not delete status")
	}

	return nil

}

func (r *gormFeatureRepository) ListFeatures(ctx context.Context, clusterID uint) ([]clusterfeature.Feature, error) {

	logger := logur.WithFields(r.logger, map[string]interface{}{"clusterID": clusterID})
	logger.Info("retrieving features for cluster...")

	var (
		featureModels []clusterFeatureModel
		featureList   []clusterfeature.Feature
	)

	if err := r.db.Find(&featureModels, clusterFeatureModel{ClusterId: clusterID}).Error; err != nil {
		logger.Debug("could not retrieve features")

		return nil, emperror.WrapWith(err, "could not retrieve features", "clusterID", clusterID)
	}

	// model  --> domain
	for _, feature := range featureModels {
		f, e := r.modelToFeature(feature)
		if e != nil {
			logger.Debug("failed to convert model to feature")
			continue
		}

		featureList = append(featureList, *f)
	}
	logger.Info("features list for cluster retrieved")
	return featureList, nil

}
