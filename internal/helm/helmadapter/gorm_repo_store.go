// Copyright © 2020 Banzai Cloud
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

package helmadapter

import (
	"context"
	"time"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/internal/helm"
)

// repositoryModel describes the helm repository model.
type repositoryModel struct {
	ID               uint `gorm:"primary_key"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	OrganizationID   uint   `gorm:"unique_index:idx_org_name"`
	Name             string `gorm:"unique_index:idx_org_name"`
	URL              string
	PasswordSecretID string
	TlsSecretID      string
}

// TableName changes the default table name.
func (repositoryModel) TableName() string {
	return "helm_repositories"
}

type helmRepoStore struct {
	db     *gorm.DB
	logger Logger
}

func NewHelmRepoStore(db *gorm.DB, logger Logger) helm.Store {
	return helmRepoStore{
		db:     db,
		logger: logger,
	}
}

func (h helmRepoStore) Delete(_ context.Context, organizationID uint, repository helm.Repository) error {
	model := toModel(organizationID, repository)

	// delete the record permanently in order for the unique constraint to be working
	if err := h.db.Unscoped().Delete(model).Error; err != nil {
		return errors.WrapIf(err, "failed to delete repository record")
	}

	h.logger.Debug("deleted helm repository record",
		map[string]interface{}{"organizationID": organizationID, "repoName": repository.Name})

	return nil
}

func (h helmRepoStore) List(_ context.Context, organizationID uint) ([]helm.Repository, error) {
	var repoModels []repositoryModel
	if err := h.db.Where("organization_id = ?", organizationID).Find(&repoModels).Error; err != nil {
		return nil, errors.WrapIf(err, "failed to list helm repositories")
	}

	repos := make([]helm.Repository, 0, len(repoModels))
	for _, model := range repoModels {
		repos = append(repos, toDomain(model))
	}

	h.logger.Debug(
		"retrieved helm repository records",
		map[string]interface{}{
			"organizationID":  organizationID,
			"repositoryCount": len(repos)})

	return repos, nil
}

func (h helmRepoStore) Create(_ context.Context, organizationID uint, repository helm.Repository) error {
	repoModel := toModel(organizationID, repository)

	if err := h.db.Create(&repoModel).Error; err != nil {
		return errors.WrapIf(err, "failed to persist the helm repository")
	}

	h.logger.Debug(
		"persisted new helm repository record",
		map[string]interface{}{
			"organizationID": organizationID,
			"repoName":       repository.Name})

	return nil
}

func (h helmRepoStore) Get(_ context.Context, organizationID uint, repository helm.Repository) (helm.Repository, error) {
	repoModel := toModel(organizationID, repository)

	if err := h.db.Where(&repoModel).First(&repoModel).Error; err != nil {
		return helm.Repository{}, errors.WrapIf(err, "failed to get helm repository")
	}
	h.logger.Debug("retrieved helm repository record",
		map[string]interface{}{"organizationID": organizationID, "repoName": repository.Name})

	return toDomain(repoModel), nil
}

func (h helmRepoStore) Update(ctx context.Context, organizationID uint, repository helm.Repository) error {
	repoModel := toModel(organizationID, repository)

	if err := h.db.Update(&repoModel).Error; err != nil {
		return errors.WrapIfWithDetails(err, "failed to update the helm repository",
			"orgID", organizationID, "repoName", repoModel.Name)
	}

	h.logger.Debug(
		"updated helm repository record",
		map[string]interface{}{
			"organizationID": organizationID,
			"repoName":       repository.Name})

	return nil

}

// toDomain transforms a gorm model to a domain struct
func toDomain(model repositoryModel) helm.Repository {
	return helm.Repository{
		Name:             model.Name,
		URL:              model.URL,
		PasswordSecretID: model.PasswordSecretID,
		TlsSecretID:      model.TlsSecretID,
	}
}

//toModel transforms a domain struct to gorm model representation
func toModel(orgID uint, repository helm.Repository) repositoryModel {
	return repositoryModel{
		OrganizationID:   orgID,
		Name:             repository.Name,
		URL:              repository.URL,
		PasswordSecretID: repository.PasswordSecretID,
		TlsSecretID:      repository.TlsSecretID,
	}
}
