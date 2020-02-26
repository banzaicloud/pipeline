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

package integratedserviceadapter

import (
	"context"
	"database/sql/driver"
	"fmt"
	"time"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"
	"logur.dev/logur"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/database/sql/json"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
)

// TableName constants
const (
	integratedServiceTableName = "cluster_features"
)

type integratedServiceSpec map[string]interface{}

func (fs *integratedServiceSpec) Scan(src interface{}) error {
	return json.Scan(src, fs)
}

func (fs integratedServiceSpec) Value() (driver.Value, error) {
	return json.Value(fs)
}

// integratedServiceModel describes the cluster group model.
type integratedServiceModel struct {
	// injecting timestamp fields
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Name      string `gorm:"unique_index:idx_cluster_feature_cluster_id_name"`
	Status    string
	ClusterId uint                  `gorm:"unique_index:idx_cluster_feature_cluster_id_name"`
	Spec      integratedServiceSpec `gorm:"type:text"`
	CreatedBy uint
}

// TableName changes the default table name.
func (cfm integratedServiceModel) TableName() string {
	return integratedServiceTableName
}

// String method prints formatted cluster fields.
func (cfm integratedServiceModel) String() string {
	return fmt.Sprintf("Id: %d, Creation date: %s, Name: %s", cfm.ID, cfm.CreatedAt, cfm.Name)
}

// GORMIntegratedServiceRepository implements integrated service persistence in RDBMS using GORM.
// TODO: write integration tests
type GORMIntegratedServiceRepository struct {
	db     *gorm.DB
	logger common.Logger
}

// NewGormIntegratedServiceRepository returns an integrated service repository persisting integrated service state into database using GORM.
func NewGormIntegratedServiceRepository(db *gorm.DB, logger common.Logger) GORMIntegratedServiceRepository {
	return GORMIntegratedServiceRepository{
		db:     db,
		logger: logger,
	}
}

// GetIntegratedServices returns integrated services stored in the repository for the specified cluster.
func (r GORMIntegratedServiceRepository) GetIntegratedServices(ctx context.Context, clusterID uint) ([]integratedservices.IntegratedService, error) {
	logger := logur.WithFields(r.logger, map[string]interface{}{"clusterID": clusterID})
	logger.Info("retrieving integrated services for cluster")

	var (
		integratedServiceModels []integratedServiceModel
		integratedServiceList   []integratedservices.IntegratedService
	)

	if err := r.db.Find(&integratedServiceModels, integratedServiceModel{ClusterId: clusterID}).Error; err != nil {
		logger.Debug("could not retrieve integrated services")

		return nil, errors.WrapIfWithDetails(err, "could not retrieve integrated services", "clusterID", clusterID)
	}

	// model  --> domain
	for _, integratedService := range integratedServiceModels {
		f, e := r.modelToIntegratedService(integratedService)
		if e != nil {
			logger.Debug("failed to convert model to integrated service")
			continue
		}

		integratedServiceList = append(integratedServiceList, f)
	}

	logger.Info("integrated services list for cluster retrieved")

	return integratedServiceList, nil
}

// SaveIntegratedService persists an integrated service with the specified properties in the database.
func (r GORMIntegratedServiceRepository) SaveIntegratedService(ctx context.Context, clusterID uint, integratedServiceName string, spec integratedservices.IntegratedServiceSpec, status string) error {
	model := integratedServiceModel{
		ClusterId: clusterID,
		Name:      integratedServiceName,
	}

	if err := r.db.Where(&model).First(&model).Error; err != nil && !gorm.IsRecordNotFoundError(err) {
		return errors.WrapIfWithDetails(err, "failed to query integrated service", "clusterId", clusterID, "integrated service", integratedServiceName)
	}
	model.Spec = spec
	model.Status = status
	return errors.WrapIfWithDetails(r.db.Save(&model).Error, "failed to save integrated service", "clusterId", clusterID, "integrated service", integratedServiceName)
}

// GetIntegratedService retrieves an integrated service by the cluster ID and integrated service name.
// It returns a "integrated service not found" error if the integrated service is not in the database.
func (r GORMIntegratedServiceRepository) GetIntegratedService(ctx context.Context, clusterID uint, integratedServiceName string) (integratedservices.IntegratedService, error) {
	fm := integratedServiceModel{}

	err := r.db.First(&fm, integratedServiceModel{Name: integratedServiceName, ClusterId: clusterID}).Error

	if gorm.IsRecordNotFoundError(err) {
		return integratedservices.IntegratedService{}, integratedServiceNotFoundError{
			ClusterID:             clusterID,
			IntegratedServiceName: integratedServiceName,
		}
	} else if err != nil {
		return integratedservices.IntegratedService{}, errors.WrapIf(err, "could not retrieve integrated service")
	}

	return r.modelToIntegratedService(fm)
}

// UpdateIntegratedServiceStatus sets the status of the specified integrated service
func (r GORMIntegratedServiceRepository) UpdateIntegratedServiceStatus(ctx context.Context, clusterID uint, integratedServiceName string, status string) error {
	fm := integratedServiceModel{
		ClusterId: clusterID,
		Name:      integratedServiceName,
	}

	return errors.WrapIf(r.db.Find(&fm, fm).Updates(integratedServiceModel{Status: status}).Error, "could not update integrated service status")
}

// UpdateIntegratedServiceSpec sets the specification of the specified integrated service
func (r GORMIntegratedServiceRepository) UpdateIntegratedServiceSpec(ctx context.Context, clusterID uint, integratedServiceName string, spec integratedservices.IntegratedServiceSpec) error {
	fm := integratedServiceModel{ClusterId: clusterID, Name: integratedServiceName}

	return errors.WrapIf(r.db.Find(&fm, fm).Updates(integratedServiceModel{Spec: spec}).Error, "could not update integrated service spec")
}

func (r GORMIntegratedServiceRepository) modelToIntegratedService(cfm integratedServiceModel) (integratedservices.IntegratedService, error) {
	f := integratedservices.IntegratedService{
		Name:   cfm.Name,
		Status: cfm.Status,
		Spec:   cfm.Spec,
	}

	return f, nil
}

// DeleteIntegratedService permanently deletes the integrated service record
func (r GORMIntegratedServiceRepository) DeleteIntegratedService(ctx context.Context, clusterID uint, integratedServiceName string) error {
	fm := integratedServiceModel{ClusterId: clusterID, Name: integratedServiceName}

	if err := r.db.Delete(&fm, fm).Error; err != nil {
		return errors.WrapIf(err, "could not delete status")
	}

	return nil
}

type integratedServiceNotFoundError struct {
	ClusterID             uint
	IntegratedServiceName string
}

func (e integratedServiceNotFoundError) Error() string {
	return fmt.Sprintf("IntegratedService %q not found for cluster %d", e.IntegratedServiceName, e.ClusterID)
}

func (e integratedServiceNotFoundError) Details() []interface{} {
	return []interface{}{
		"clusterId", e.ClusterID,
		"integrated service", e.IntegratedServiceName,
	}
}

func (integratedServiceNotFoundError) IntegratedServiceNotFound() bool {
	return true
}

// NotFound tells a client that this error is related to a resource being not found.
// Can be used to translate the error to eg. status code.
func (integratedServiceNotFoundError) NotFound() bool {
	return true
}

// ServiceError tells the transport layer whether this error should be translated into the transport format
// or an internal error should be returned instead.
func (integratedServiceNotFoundError) ServiceError() bool {
	return true
}
