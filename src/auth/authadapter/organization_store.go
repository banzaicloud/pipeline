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
	"github.com/aokoli/goutils"
	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/src/auth"
)

// GormOrganizationStore implements organization membership persistence using Gorm.
type GormOrganizationStore struct {
	db     *gorm.DB
	strgen RandomStringGenerator
}

// RandomStringGenerator generates random strings.
type RandomStringGenerator interface {
	// RandomAlphabetic creates a random string whose length is the number of characters specified.
	RandomAlphabetic(count int) (string, error)
}

type GoutilsRandomStringGenerator struct{}

func (GoutilsRandomStringGenerator) RandomAlphabetic(count int) (string, error) {
	return goutils.RandomAlphabetic(count)
}

// NewGormOrganizationStore returns a new GormOrganizationStore.
func NewGormOrganizationStore(db *gorm.DB, strgen RandomStringGenerator) GormOrganizationStore {
	return GormOrganizationStore{
		db:     db,
		strgen: strgen,
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
		normalizedName := NormalizeOrganizationName(name)

		var normalizedNameCount int
		if err := g.db.Where(auth.Organization{NormalizedName: normalizedName}).Model(auth.Organization{}).Count(&normalizedNameCount).Error; err != nil {
			return false, 0, errors.WrapIfWithDetails(
				err,
				"failed to check organization name",
				"organizationName", name,
				"normalizedName", normalizedName,
			)
		}

		// Name is already taken
		if normalizedNameCount > 0 {
			random, err := g.strgen.RandomAlphabetic(6)
			if err != nil {
				return false, 0, errors.WrapIfWithDetails(
					err,
					"failed to generate normalized organization name",
					"organizationName", name,
					"normalizedName", normalizedName,
				)
			}

			normalizedName = normalizedName + "-" + random
		}

		organization := auth.Organization{
			Name:           name,
			NormalizedName: normalizedName,
			Provider:       provider,
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

	return false, organization.ID, nil
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

// FindUserRole returns the user's role in a given organization.
// Returns false as the second parameter if the user is not a member of the organization.
func (g GormOrganizationStore) FindUserRole(ctx context.Context, orgID uint, userID uint) (string, bool, error) {
	var membership auth.UserOrganization

	err := g.db.Where(auth.UserOrganization{UserID: userID, OrganizationID: orgID}).First(&membership).Error
	if gorm.IsRecordNotFoundError(err) {
		return "", false, nil
	} else if err != nil {
		return "", false, errors.WrapIfWithDetails(
			err, "cannot fetch organization membership details from the database",
			"organizationId", orgID,
			"userId", userID,
		)
	}

	return membership.Role, true, nil
}
