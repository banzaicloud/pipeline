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

package authadapter

import (
	"context"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/auth"
)

// GormOrganizationStore implements organization membership persistence using Gorm.
type GormOrganizationStore struct {
	db *gorm.DB
}

// NewGormOrganizationStore returns a new GormOrganizationStore.
func NewGormOrganizationStore(db *gorm.DB) GormOrganizationStore {
	return GormOrganizationStore{
		db: db,
	}
}

// EnsureOrganizationExists ensures that an organization exists.
func (g GormOrganizationStore) EnsureOrganizationExists(ctx context.Context, name string, provider string) (bool, uint, error) {
	var organization auth.Organization

	err := g.db.
		Where(auth.Organization{Name: name}).
		First(&organization).
		Error
	if gorm.IsRecordNotFoundError(err) {
		organization := auth.Organization{
			Name:     name,
			Provider: provider,
		}

		err := g.db.Save(&organization).Error
		if err != nil {
			return false, 0, errors.WrapIfWithDetails(
				err,
				"failed to create organization",
				"organizationName", name,
			)
		}

		return true, organization.ID, nil
	}
	if err != nil {
		return false, 0, errors.WrapIfWithDetails(
			err,
			"failed to get organization",
			"organizationName", name,
		)
	}

	if organization.Provider != provider {
		return false, 0, errors.WithDetails(
			errors.WithStack(auth.ErrOrganizationConflict),
			"organizationName", name,
		)
	}

	return false, 0, nil
}

// GetOrganizationMembershipsOf returns the list of organization memberships for a user.
func (g GormOrganizationStore) GetOrganizationMembershipsOf(ctx context.Context, userID uint) ([]auth.UserOrganization, error) {
	var memberships []auth.UserOrganization

	err := g.db.
		Preload("Organization").
		Find(&memberships, auth.UserOrganization{UserID: userID}).
		Error
	if err != nil {
		return nil, errors.WrapIfWithDetails(
			err,
			"failed to get memberships for user",
			"userId", userID,
		)
	}

	return memberships, nil
}

// RemoveUserFromOrganization removes a user from an organization.
func (g GormOrganizationStore) RemoveUserFromOrganization(ctx context.Context, organizationID uint, userID uint) error {
	err := g.db.
		Model(auth.User{ID: userID}).
		Association("Organizations").
		Delete(auth.Organization{ID: organizationID}).
		Error
	if err != nil {
		return errors.WrapIfWithDetails(
			err,
			"failed to remove user from organization",
			"userId", userID,
			"organizationId", organizationID,
		)
	}

	return nil
}

// ApplyUserMembership ensures that a user is a member of an organization with the necessary role.
func (g GormOrganizationStore) ApplyUserMembership(ctx context.Context, organizationID uint, userID uint, role string) error {
	var userOrganization auth.UserOrganization

	err := g.db.
		Where(auth.UserOrganization{UserID: userID, OrganizationID: organizationID}).
		First(&userOrganization).
		Error
	if gorm.IsRecordNotFoundError(err) {
		userOrganization := auth.UserOrganization{
			UserID:         userID,
			OrganizationID: organizationID,
			Role:           role,
		}

		err = g.db.Save(&userOrganization).Error
		if err != nil {
			return errors.WrapIfWithDetails(
				err,
				"failed to apply user membership",
				"userId", userID,
				"organizationId", organizationID,
				"role", role,
			)
		}
	}
	if err != nil {
		return errors.WrapIfWithDetails(
			err,
			"failed to get current user membership",
			"userId", userID,
			"organizationId", organizationID,
			"role", role,
		)
	}

	err = g.db.
		Model(&auth.UserOrganization{}).
		Where(auth.UserOrganization{UserID: userID, OrganizationID: organizationID}).
		Update(auth.UserOrganization{Role: role}).
		Error
	if err != nil {
		return errors.WrapIfWithDetails(
			err,
			"failed to apply user membership",
			"userId", userID,
			"organizationId", organizationID,
			"role", role,
		)
	}

	return nil
}
