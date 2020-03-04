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
	"fmt"
	"testing"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/helm"
)

func setUpDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open("sqlite3", "file::memory:")
	require.NoError(t, err)

	err = Migrate(db, common.NoopLogger{})
	require.NoError(t, err)

	return db
}

func Test_helmRepoStore_Create(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		db := setUpDatabase(t)
		store := NewHelmRepoStore(db, common.NoopLogger{})

		newRepo := helm.Repository{
			Name:             "testing",
			URL:              "repoURL",
			PasswordSecretID: "secretRef",
		}

		err := store.Create(context.Background(), 1, newRepo)
		require.NoError(t, err)

		retrieved, err := store.Get(context.Background(), 1, newRepo)

		require.NoError(t, err)
		assert.Equal(t, newRepo.Name, retrieved.Name)
	})

	t.Run("AlreadyExists", func(t *testing.T) {
		db := setUpDatabase(t)
		store := NewHelmRepoStore(db, common.NoopLogger{})

		newRepo := helm.Repository{
			Name:             "violation",
			URL:              "repoURL",
			PasswordSecretID: "secretRef",
		}

		err := store.Create(context.Background(), 1, newRepo)
		require.NoError(t, err)

		err = store.Create(context.Background(), 1, newRepo)
		// addition fails due to constraint violation
		require.Error(t, err)
	})
}

func Test_helmRepoStore_Get(t *testing.T) {
	t.Run("NotFound", func(t *testing.T) {
		db := setUpDatabase(t)
		store := NewHelmRepoStore(db, common.NoopLogger{})

		newRepo := helm.Repository{
			Name:             "testing",
			URL:              "repoURL",
			PasswordSecretID: "secretRef",
		}

		_, err := store.Get(context.Background(), 1, newRepo)
		require.Error(t, err)
	})

	t.Run("Success", func(t *testing.T) {
		db := setUpDatabase(t)
		store := NewHelmRepoStore(db, common.NoopLogger{})

		newRepo := helm.Repository{
			Name:             "testing",
			URL:              "repoURL",
			PasswordSecretID: "secretRef",
		}

		err := store.Create(context.Background(), 1, newRepo)
		require.NoError(t, err)

		retrieved, err := store.Get(context.Background(), 1, newRepo)
		require.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, retrieved, newRepo)
	})
}

func Test_helmRepoStore_Delete(t *testing.T) {
	t.Run("DoesntExist", func(t *testing.T) {
		db := setUpDatabase(t)
		store := NewHelmRepoStore(db, common.NoopLogger{})

		toBeDeleted := helm.Repository{
			Name:             "testing",
			URL:              "repoURL",
			PasswordSecretID: "secretRef",
		}

		err := store.Delete(context.Background(), 1, toBeDeleted)
		require.NoError(t, err)
	})

	t.Run("Success", func(t *testing.T) {
		db := setUpDatabase(t)
		store := NewHelmRepoStore(db, common.NoopLogger{})

		toBeDeleted := helm.Repository{
			Name:             "testing",
			URL:              "repoURL",
			PasswordSecretID: "secretRef",
		}

		err := store.Create(context.Background(), 1, toBeDeleted)
		require.NoError(t, err)

		err = store.Delete(context.Background(), 1, toBeDeleted)
		require.NoError(t, err)
	})
}

func Test_helmRepoStore_ListRepositories(t *testing.T) {
	t.Run("NoneFound", func(t *testing.T) {
		db := setUpDatabase(t)
		store := NewHelmRepoStore(db, common.NoopLogger{})

		repos, err := store.List(context.Background(), 1)
		require.NoError(t, err)
		require.NotNil(t, repos)
	})

	t.Run("Success", func(t *testing.T) {
		db := setUpDatabase(t)
		store := NewHelmRepoStore(db, common.NoopLogger{})

		for i := 0; i < 3; i++ {
			if err := store.Create(context.Background(), 1, helm.Repository{
				Name:             fmt.Sprintf("list-%d", i),
				URL:              "repoURL",
				PasswordSecretID: "secretRef",
			}); err != nil {
				t.Fatal("failed to create repository")
			}
		}

		repos, err := store.List(context.Background(), 1)
		require.NoError(t, err)
		require.NotNil(t, repos)
		assert.Equal(t, 3, len(repos))
	})
}

func Test_helmRepoStore_Update(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		db := setUpDatabase(t)
		store := NewHelmRepoStore(db, common.NoopLogger{})

		newRepo := helm.Repository{
			Name:             "testing",
			URL:              "repoURL",
			PasswordSecretID: "secretRef",
		}

		err := store.Create(context.Background(), 1, newRepo)
		require.NoError(t, err)

		retrieved, err := store.Get(context.Background(), 1, newRepo)

		require.NoError(t, err)
		assert.Equal(t, newRepo.Name, retrieved.Name)

		updatedRepo := helm.Repository{
			Name:             "testing",
			URL:              "UpdatedrepoURL",
			PasswordSecretID: "UpdatedsecretRef",
		}

		err = store.Update(context.Background(), 1, updatedRepo)
		require.NoError(t, err)

		retrieved, err = store.Get(context.Background(), 1, updatedRepo)

		require.NoError(t, err)
		assert.Equal(t, updatedRepo.Name, retrieved.Name)
		assert.Equal(t, updatedRepo.URL, retrieved.URL)
		assert.Equal(t, updatedRepo.PasswordSecretID, retrieved.PasswordSecretID)
	})
}
