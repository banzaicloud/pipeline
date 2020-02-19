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

package helm3

import (
	"context"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/internal/common"
)

type helmRepoStore struct {
	db     *gorm.DB
	logger common.Logger
}

func NewHelmRepoStore(db *gorm.DB, logger common.Logger) Store {
	return helmRepoStore{
		db:     db,
		logger: logger,
	}
}

func (h helmRepoStore) DeleteRepository(_ context.Context, organizationID uint, repository Repository) error {
	model := ToModel(repository)
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

func (h helmRepoStore) ListRepositories(_ context.Context, organizationID uint) ([]Repository, error) {
	var repoModels []RepositoryModel
	if err := h.db.Where("organization_id = ?", organizationID).Find(&repoModels).Error; err != nil {
		return nil, errors.WrapIf(err, "failed to list helm repositories")
	}

	repos := make([]Repository, 0, len(repoModels))
	for _, model := range repoModels {
		repos = append(repos, ToDomain(model))
	}

	h.logger.Debug("retrieved helm repository records",
		map[string]interface{}{"organisationID": organizationID, "repositories #": len(repos)})

	return repos, nil
}

func (h helmRepoStore) AddRepository(_ context.Context, organizationID uint, repository Repository) error {
	repoModel := ToModel(repository)
	repoModel.OrganizationID = organizationID

	if err := h.db.Create(&repoModel).Error; err != nil {
		return errors.WrapIf(err, "failed to persist the helm repository")
	}
	h.logger.Debug("persisted new helm repository record",
		map[string]interface{}{"organisationID": organizationID, "repoName": repository.Name})

	return nil
}

// ToDomain transforms a gorm model to a domain struct
func ToDomain(model RepositoryModel) Repository {
	return Repository{
		Name:             model.Name,
		URL:              model.URL,
		PasswordSecretID: model.PasswordSecretID,
		TlsSecretID:      model.TlsSecretID,
	}
}

//ToModel transforms a domain struct to gorm model representation
func ToModel(repository Repository) RepositoryModel {
	return RepositoryModel{
		Name:             repository.Name,
		PasswordSecretID: repository.PasswordSecretID,
		TlsSecretID:      repository.TlsSecretID,
	}
}
