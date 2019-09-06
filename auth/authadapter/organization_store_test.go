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
	"errors"
	"io/ioutil"
	"testing"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite" // SQLite driver used for integration test
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/auth"
)

func setUpDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open("sqlite3", "file::memory:")
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetOutput(ioutil.Discard)

	err = auth.Migrate(db, logger)
	require.NoError(t, err)

	return db
}

func TestGormOrganizationStore_EnsureOrganizationExists(t *testing.T) {
	t.Parallel()

	t.Run("create", func(t *testing.T) {
		db := setUpDatabase(t)
		store := NewGormOrganizationStore(db)

		created, id, err := store.EnsureOrganizationExists(context.Background(), "example", "github")
		require.NoError(t, err)

		var organization auth.Organization

		err = db.
			Where(auth.Organization{Name: "example"}).
			First(&organization).
			Error
		require.NoError(t, err)

		assert.True(t, created)
		assert.Equal(t, organization.ID, id)
	})

	t.Run("already_exists", func(t *testing.T) {
		db := setUpDatabase(t)
		store := NewGormOrganizationStore(db)

		organization := auth.Organization{Name: "example", Provider: "github"}

		err := db.Save(&organization).Error
		require.NoError(t, err)

		created, id, err := store.EnsureOrganizationExists(context.Background(), "example", "github")
		require.NoError(t, err)

		assert.False(t, created)
		assert.Equal(t, uint(0), id)
	})

	t.Run("conflict", func(t *testing.T) {
		db := setUpDatabase(t)
		store := NewGormOrganizationStore(db)

		organization := auth.Organization{Name: "example", Provider: "github"}

		err := db.Save(&organization).Error
		require.NoError(t, err)

		created, id, err := store.EnsureOrganizationExists(context.Background(), "example", "gitlab")
		require.Error(t, err)

		assert.True(t, errors.Is(err, auth.ErrOrganizationConflict))
		assert.False(t, created)
		assert.Equal(t, uint(0), id)
	})
}

func TestGormOrganizationStore_GetOrganizationMembershipsOf(t *testing.T) {
	db := setUpDatabase(t)
	store := NewGormOrganizationStore(db)

	user := auth.User{
		Name:  "John Doe",
		Email: "john.doe@example.com",
		Login: "john.doe",
		Organizations: []auth.Organization{
			{
				Name:     "example",
				Provider: "github",
			},
		},
	}

	err := db.Save(&user).Error
	require.NoError(t, err)

	currentMemberships, err := store.GetOrganizationMembershipsOf(context.Background(), user.ID)
	require.NoError(t, err)

	require.Len(t, currentMemberships, 1, "user is expected to be the member of one organization")
	assert.Equal(t, user.Organizations[0].Name, currentMemberships[0].Organization.Name)
	assert.Equal(t, auth.RoleMember, currentMemberships[0].Role)
}

func TestGormOrganizationStore_RemoveUserFromOrganization(t *testing.T) {
	db := setUpDatabase(t)
	store := NewGormOrganizationStore(db)

	user := auth.User{
		Name:  "John Doe",
		Email: "john.doe@example.com",
		Login: "john.doe",
		Organizations: []auth.Organization{
			{
				Name:     "example",
				Provider: "github",
			},
			{
				Name:     "remove-from-this",
				Provider: "github",
			},
		},
	}

	err := db.Save(&user).Error
	require.NoError(t, err)

	err = store.RemoveUserFromOrganization(context.Background(), user.Organizations[1].ID, user.ID)
	require.NoError(t, err)

	var organizations []auth.Organization

	err = db.Model(user).Association("Organizations").Find(&organizations).Error

	require.Len(t, organizations, 1, "user is expected to be the member of one organization")
	assert.Equal(t, user.Organizations[0].Name, organizations[0].Name)
}

func TestGormOrganizationStore_ApplyUserMembership(t *testing.T) {
	t.Parallel()

	t.Run("existing", func(t *testing.T) {
		db := setUpDatabase(t)
		store := NewGormOrganizationStore(db)

		user := auth.User{
			Name:  "John Doe",
			Email: "john.doe@example.com",
			Login: "john.doe",
			Organizations: []auth.Organization{
				{
					Name:     "example",
					Provider: "github",
				},
			},
		}

		err := db.Save(&user).Error
		require.NoError(t, err)

		err = store.ApplyUserMembership(context.Background(), user.Organizations[0].ID, user.ID, auth.RoleAdmin)
		require.NoError(t, err)

		var userOrganization auth.UserOrganization

		err = db.
			Where(auth.UserOrganization{UserID: user.ID, OrganizationID: user.Organizations[0].ID}).
			First(&userOrganization).
			Error
		require.NoError(t, err)

		assert.Equal(t, userOrganization.Role, auth.RoleAdmin, "user is expected to be an admin")
	})

	t.Run("existing_no_change", func(t *testing.T) {
		db := setUpDatabase(t)
		store := NewGormOrganizationStore(db)

		user := auth.User{
			Name:  "John Doe",
			Email: "john.doe@example.com",
			Login: "john.doe",
			Organizations: []auth.Organization{
				{
					Name:     "example",
					Provider: "github",
				},
			},
		}

		err := db.Save(&user).Error
		require.NoError(t, err)

		err = store.ApplyUserMembership(context.Background(), user.Organizations[0].ID, user.ID, auth.RoleMember)
		require.NoError(t, err)

		var userOrganization auth.UserOrganization

		err = db.
			Where(auth.UserOrganization{UserID: user.ID, OrganizationID: user.Organizations[0].ID}).
			First(&userOrganization).
			Error
		require.NoError(t, err)

		assert.Equal(t, userOrganization.Role, auth.RoleMember, "user is expected to be a member")
	})

	t.Run("new", func(t *testing.T) {
		db := setUpDatabase(t)
		store := NewGormOrganizationStore(db)

		user := auth.User{
			Name:  "John Doe",
			Email: "john.doe@example.com",
			Login: "john.doe",
		}

		organization := auth.Organization{
			Name:     "example",
			Provider: "github",
		}

		err := db.Save(&user).Error
		require.NoError(t, err)

		err = db.Save(&organization).Error
		require.NoError(t, err)

		err = store.ApplyUserMembership(context.Background(), organization.ID, user.ID, auth.RoleAdmin)
		require.NoError(t, err)

		var userOrganization auth.UserOrganization

		err = db.
			Where(auth.UserOrganization{UserID: user.ID, OrganizationID: organization.ID}).
			First(&userOrganization).
			Error
		require.NoError(t, err)

		assert.Equal(t, userOrganization.Role, auth.RoleAdmin, "user is expected to be an admin")
	})
}
