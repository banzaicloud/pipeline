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

package helm

import (
	"emperror.dev/errors"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"k8s.io/helm/pkg/repo"
)

// HelmRepoRepository
type HelmRepoRepository struct {
	db     *gorm.DB
	logger logrus.FieldLogger
}

// NewHelmRepoRepository returns a new HelmRepoRepository instance.
func NewHelmRepoRepository(
	db *gorm.DB,
	logger logrus.FieldLogger,
) *HelmRepoRepository {
	return &HelmRepoRepository{
		db:     db,
		logger: logger,
	}
}

// FindOne returns a Helm repo instance for an organization by repo name.
func (g *HelmRepoRepository) FindOne(orgID uint, repoName string) (*HelmRepoModel, error) {
	var repo HelmRepoModel
	repo.OrgID = orgID
	repo.Name = repoName
	err := g.db.Where(repo).First(&repo).Error
	if gorm.IsRecordNotFoundError(err) {
		return nil, errors.WrapWithDetails(errors.New("no Helm repo found"), "orgID", repo.OrgID, "repoName", repoName)
	}
	if err != nil {
		return nil, errors.WrapWithDetails(err, "error finding Helm repo", "orgID", repo.OrgID, "repoName", repoName)
	}

	return &repo, nil
}

// FindAll returns all Helm repo for an orgID
func (g *HelmRepoRepository) FindAll(orgID uint) ([]*HelmRepoModel, error) {
	var cgroups []*HelmRepoModel

	err := g.db.Where(HelmRepoModel{
		OrgID: orgID,
	}).Find(&cgroups).Error
	if err != nil {
		return nil, errors.WrapWithDetails(err, "could not find Helm repos", "orgID", orgID)
	}

	return cgroups, nil
}

// Create persists a Helm repo
func (g *HelmRepoRepository) Create(orgID uint, r repo.Entry) (*uint, error) {
	repo := HelmRepoModel{
		Name:     r.Name,
		OrgID:    orgID,
		URL:      r.URL,
		Username: r.Username,
		Password: r.Password,
		CertFile: r.CertFile,
		KeyFile:  r.KeyFile,
		CAFile:   r.CAFile,
	}
	err := g.db.Save(repo).Error
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "error creating Helm repo", "orgID", repo.OrgID, "repoName", repo.Name)
	}
	return &repo.ID, nil
}

// UpdateMembers updates a Helm repo
func (g *HelmRepoRepository) Update(orgID uint, r repo.Entry) error {
	repoModel, err := g.FindOne(orgID, r.Name)
	if err != nil {
		return err
	}

	repoModel.URL = r.URL
	repoModel.Username = r.Username
	repoModel.Password = r.Password
	repoModel.CertFile = r.CertFile
	repoModel.KeyFile = r.KeyFile
	repoModel.CAFile = r.CAFile

	err = g.db.Save(repoModel).Error
	if err != nil {
		return errors.WrapIfWithDetails(err, "could not update Helm repo", "orgID", orgID, "repoName", r.Name)
	}
	return nil
}

// Delete deletes a Helm repo
func (g *HelmRepoRepository) Delete(repo *HelmRepoModel) error {
	err := g.db.Delete(repo).Error
	if err != nil {
		return errors.WrapIfWithDetails(err, "could not delete Helm repo", "orgID", repo.OrgID, "repoName", repo.Name)
	}

	return nil
}
