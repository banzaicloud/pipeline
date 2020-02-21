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

package helmrepoadapter

import (
	"context"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/helm3"
)

// repositoryModel describes the common cluster model.
type repositoryModel struct {
	gorm.Model

	Name             string
	URL              string
	OrganizationID   uint // FK to organizations
	PasswordSecretID string
	TlsSecretID      string
}

// TableName changes the default table name.
func (repositoryModel) TableName() string {
	return "helm_repositories"
}

type helmRepoStore struct {
	db     *gorm.DB
	logger common.Logger
}

func NewHelmRepoStore(db *gorm.DB, logger common.Logger) helm3.Store {
	return helmRepoStore{
		db:     db,
		logger: logger,
	}
}

func (h helmRepoStore) DeleteRepository(_ context.Context, organizationID uint, repository helm3.Repository) error {
	model := toModel(repository)
	model.OrganizationID = organizationID

	if err := h.db.Where(model).First(&model).Error; err != nil {
		return errors.WrapIf(err, "failed to load helm repository record")
	}

	if err := h.db.Delete(model).Error; err != nil {
		return errors.WrapIf(err, "failed to delete repository record")
	}

	h.logger.Debug("deleted helm repository record",
		map[string]interface{}{"organisationID": organizationID, "repoName": repository.Name})

	return nil
}

func (h helmRepoStore) ListRepositories(_ context.Context, organizationID uint) ([]helm3.Repository, error) {
	var repoModels []repositoryModel
	if err := h.db.Where("organization_id = ?", organizationID).Find(&repoModels).Error; err != nil {
		return nil, errors.WrapIf(err, "failed to list helm repositories")
	}

	repos := make([]helm3.Repository, 0, len(repoModels))
	for _, model := range repoModels {
		repos = append(repos, toDomain(model))
	}

	h.logger.Debug("retrieved helm repository records",
		map[string]interface{}{"organisationID": organizationID, "repositories #": len(repos)})

	return repos, nil
}

func (h helmRepoStore) AddRepository(_ context.Context, organizationID uint, repository helm3.Repository) error {
	repoModel := toModel(repository)
	repoModel.OrganizationID = organizationID

	if err := h.db.Create(&repoModel).Error; err != nil {
		return errors.WrapIf(err, "failed to persist the helm repository")
	}
	h.logger.Debug("persisted new helm repository record",
		map[string]interface{}{"organisationID": organizationID, "repoName": repository.Name})

	return nil
}

func (h helmRepoStore) GetRepository(_ context.Context, organizationID uint, repository helm3.Repository) (helm3.Repository, error) {
	repoModel := toModel(repository)
	repoModel.OrganizationID = organizationID

	if err := h.db.Where(&repoModel).First(&repoModel).Error; err != nil {
		return helm3.Repository{}, errors.WrapIf(err, "failed to list helm repositories")
	}
	h.logger.Debug("retrieved helm repository record",
		map[string]interface{}{"organisationID": organizationID, "repoName": repository.Name})

	return toDomain(repoModel), nil
}

// toDomain transforms a gorm model to a domain struct
func toDomain(model repositoryModel) helm3.Repository {
	return helm3.Repository{
		Name:             model.Name,
		URL:              model.URL,
		PasswordSecretID: model.PasswordSecretID,
		TlsSecretID:      model.TlsSecretID,
	}
}

//toModel transforms a domain struct to gorm model representation
func toModel(repository helm3.Repository) repositoryModel {
	return repositoryModel{
		Name:             repository.Name,
		URL:              repository.URL,
		PasswordSecretID: repository.PasswordSecretID,
		TlsSecretID:      repository.TlsSecretID,
	}
}
