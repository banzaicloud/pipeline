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

package ark

import (
	"emperror.dev/emperror"
	arkAPI "github.com/heptio/ark/pkg/apis/ark/v1"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/ark/api"
)

// RestoresService is for managing ARK restores
type RestoresService struct {
	deployments *DeploymentsService
	repository  *RestoresRepository

	org    *auth.Organization
	logger logrus.FieldLogger
}

// RestoresServiceFactory creates and returns an initialized RestoresService instance
func RestoresServiceFactory(
	org *auth.Organization,
	deployments *DeploymentsService,
	db *gorm.DB,
	logger logrus.FieldLogger,
) *RestoresService {

	return NewRestoresService(org, deployments, NewRestoresRepository(org, deployments.GetCluster(), db, logger), logger)
}

// NewRestoresService creates and returns an initialized RestoresService instance
func NewRestoresService(
	org *auth.Organization,
	deployments *DeploymentsService,
	repository *RestoresRepository,
	logger logrus.FieldLogger,
) *RestoresService {
	return &RestoresService{
		org:         org,
		deployments: deployments,
		repository:  repository,
		logger:      logger,
	}
}

// GetModelByName gets a ClusterBackupRestoresModel by name
func (s *RestoresService) GetModelByName(name string) (*ClusterBackupRestoresModel, error) {

	model, err := s.repository.FindOneByName(name)
	if err != nil {
		return nil, errors.Wrap(err, "could not get restore from database")
	}

	return model, nil
}

// GetByName gets a Restore by name
func (s *RestoresService) GetByName(name string) (*api.Restore, error) {

	model, err := s.GetModelByName(name)
	if err != nil {
		return nil, err
	}

	return model.ConvertModelToEntity(), nil
}

// GetModelByID gets a ClusterBackupRestoresModel by ID
func (s *RestoresService) GetModelByID(id uint) (*ClusterBackupRestoresModel, error) {

	model, err := s.repository.FindOneByID(id)
	if err != nil {
		return nil, errors.Wrap(err, "could not get restore from database")
	}

	return model, nil
}

// GetByID gets a Restore by ID
func (s *RestoresService) GetByID(id uint) (*api.Restore, error) {

	model, err := s.GetModelByID(id)
	if err != nil {
		return nil, err
	}

	return model.ConvertModelToEntity(), nil
}

// DeleteByName deletes a Restore by name
func (s *RestoresService) DeleteByName(name string) error {

	client, err := s.deployments.GetClient()
	if err != nil {
		return emperror.Wrap(err, "error getting ark client")
	}

	err = client.DeleteRestoreByName(name)
	if k8serrors.IsNotFound(err) {
		err = nil
	}
	if err != nil {
		return emperror.Wrap(err, "error during deleting restore")
	}

	restore, err := s.GetModelByName(name)
	if err != nil {
		return emperror.Wrap(err, "error during deleting restore")
	}

	err = s.repository.Delete(restore)
	if err != nil {
		return emperror.Wrap(err, "error during deleting restore")
	}

	return nil
}

// DeleteByID deletes a Restore by ID
func (s *RestoresService) DeleteByID(id uint) error {

	restore, err := s.GetModelByID(id)
	if err != nil {
		return emperror.Wrap(err, "could not get restore from database")
	}

	client, err := s.deployments.GetClient()
	if err != nil {
		return emperror.Wrap(err, "could not get ARK client")
	}

	err = client.DeleteRestoreByName(restore.Name)
	if k8serrors.IsNotFound(err) {
		err = nil
	}
	if err != nil {
		return emperror.Wrap(err, "could not delete restore through ARK")
	}

	err = s.repository.Delete(restore)
	if err != nil {
		return emperror.Wrap(err, "could not delete restore from database")
	}

	return nil
}

// ListFromARK gets restores from ARK
func (s *RestoresService) ListFromARK() ([]arkAPI.Restore, error) {

	client, err := s.deployments.GetClient()
	if err != nil {
		return nil, emperror.Wrap(err, "error getting ark client")
	}

	var listOptions metav1.ListOptions
	restores, err := client.ListRestores(listOptions)
	if err != nil {
		return nil, emperror.Wrap(err, "error getting restores")
	}

	return restores.Items, nil
}

// List gets all restores stored in the DB
func (s *RestoresService) List() ([]*api.Restore, error) {

	restores := make([]*api.Restore, 0)

	items, err := s.repository.Find()
	if err != nil {
		return restores, err
	}

	for _, item := range items {
		restore := item.ConvertModelToEntity()
		restores = append(restores, restore)
	}

	return restores, nil
}

// Create creates and persists a restore by a CreateRestoreRequest
func (s *RestoresService) Create(req api.CreateRestoreRequest) (*api.Restore, error) {

	deployment, err := s.deployments.GetActiveDeployment()
	if err != nil {
		return nil, emperror.Wrap(err, "error getting active deployment")
	}

	client, err := s.deployments.GetClient()
	if err != nil {
		return nil, emperror.Wrap(err, "error getting ark client")
	}

	restore, err := client.CreateRestore(req)
	if err != nil {
		return nil, emperror.Wrap(err, "error creating restore")
	}

	if restore.Status.Phase == "" {
		restore.Status.Phase = "Creating"
	}

	r := &api.PersistRestoreRequest{
		BucketID:  deployment.BucketID,
		ClusterID: s.deployments.GetCluster().GetID(),

		Restore: restore,
	}

	restoreItem, err := s.Persist(r)
	if err != nil {
		return nil, emperror.Wrap(err, "error persisting restore")
	}

	return restoreItem, nil
}

// Persist persists a restore by a PersistRestoreRequest
func (s *RestoresService) Persist(req *api.PersistRestoreRequest) (*api.Restore, error) {

	restore, err := s.repository.Persist(req)
	if err != nil {
		return nil, err
	}

	return restore.ConvertModelToEntity(), nil
}
