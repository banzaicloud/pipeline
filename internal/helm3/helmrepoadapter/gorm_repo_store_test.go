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
	"io/ioutil"
	"testing"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"logur.dev/logur"

	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	"github.com/banzaicloud/pipeline/internal/helm3"
)

func setUpDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open("sqlite3", "file::memory:")
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetOutput(ioutil.Discard)

	err = Migrate(db, logger)
	require.NoError(t, err)

	return db
}

func Test_helmRepoStore_AddRepository(t *testing.T) {
	t.Run("create - success", func(t *testing.T) {
		db := setUpDatabase(t)
		store := NewHelmRepoStore(db, commonadapter.NewLogger(logur.NoopLogger{}))

		newRepo := helm3.Repository{
			Name:             "testing",
			URL:              "repoURL",
			PasswordSecretID: "secretRef",
		}

		err := store.AddRepository(context.Background(), 1, newRepo)
		require.NoError(t, err)

		retrieved, err := store.GetRepository(context.Background(), 1, newRepo)

		require.NoError(t, err)
		assert.Equal(t, newRepo.Name, retrieved.Name)
	})

	t.Run("create - unique constraint violation", func(t *testing.T) {
		db := setUpDatabase(t)
		store := NewHelmRepoStore(db, commonadapter.NewLogger(logur.NoopLogger{}))

		newRepo := helm3.Repository{
			Name:             "violation",
			URL:              "repoURL",
			PasswordSecretID: "secretRef",
		}

		err := store.AddRepository(context.Background(), 1, newRepo)
		require.NoError(t, err)

		err = store.AddRepository(context.Background(), 1, newRepo)
		// addition fails due to constraint violation
		require.Error(t, err)
	})

}

func Test_helmRepoStore_GetRepository(t *testing.T) {
	t.Run("get repository - not found", func(t *testing.T) {
		db := setUpDatabase(t)
		store := NewHelmRepoStore(db, commonadapter.NewLogger(logur.NoopLogger{}))

		newRepo := helm3.Repository{
			Name:             "testing",
			URL:              "repoURL",
			PasswordSecretID: "secretRef",
		}

		_, err := store.GetRepository(context.Background(), 1, newRepo)
		require.Error(t, err)
	})

	t.Run("get repository - success", func(t *testing.T) {
		db := setUpDatabase(t)
		store := NewHelmRepoStore(db, commonadapter.NewLogger(logur.NoopLogger{}))

		newRepo := helm3.Repository{
			Name:             "testing",
			URL:              "repoURL",
			PasswordSecretID: "secretRef",
		}

		err := store.AddRepository(context.Background(), 1, newRepo)
		require.NoError(t, err)

		retrieved, err := store.GetRepository(context.Background(), 1, newRepo)
		require.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, retrieved, newRepo)
	})
}

func Test_helmRepoStore_DeleteRepository(t *testing.T) {
	t.Run("delete repository - not exists", func(t *testing.T) {
		db := setUpDatabase(t)
		store := NewHelmRepoStore(db, commonadapter.NewLogger(logur.NoopLogger{}))

		toBeDeleted := helm3.Repository{
			Name:             "testing",
			URL:              "repoURL",
			PasswordSecretID: "secretRef",
		}

		err := store.DeleteRepository(context.Background(), 1, toBeDeleted)
		require.Error(t, err)
	})

	t.Run("delete repository - success", func(t *testing.T) {
		db := setUpDatabase(t)
		store := NewHelmRepoStore(db, commonadapter.NewLogger(logur.NoopLogger{}))

		toBeDeleted := helm3.Repository{
			Name:             "testing",
			URL:              "repoURL",
			PasswordSecretID: "secretRef",
		}

		err := store.AddRepository(context.Background(), 1, toBeDeleted)
		require.NoError(t, err)

		err = store.DeleteRepository(context.Background(), 1, toBeDeleted)
		require.NoError(t, err)
	})
}

func Test_helmRepoStore_ListRepositories(t *testing.T) {
	t.Run("list repositories - none found", func(t *testing.T) {
		db := setUpDatabase(t)
		store := NewHelmRepoStore(db, commonadapter.NewLogger(logur.NoopLogger{}))

		repos, err := store.ListRepositories(context.Background(), 1)
		require.NoError(t, err)
		require.NotNil(t, repos)
	})

	t.Run("list repositories - success", func(t *testing.T) {
		db := setUpDatabase(t)
		store := NewHelmRepoStore(db, commonadapter.NewLogger(logur.NoopLogger{}))

		store.AddRepository(context.Background(), 1, helm3.Repository{
			Name:             "list-0",
			URL:              "repoURL",
			PasswordSecretID: "secretRef",
		})
		store.AddRepository(context.Background(), 1, helm3.Repository{
			Name:             "list-2",
			URL:              "repoURL",
			PasswordSecretID: "secretRef",
		})
		store.AddRepository(context.Background(), 1, helm3.Repository{
			Name:             "list-3",
			URL:              "repoURL",
			PasswordSecretID: "secretRef",
		})

		repos, err := store.ListRepositories(context.Background(), 1)
		require.NoError(t, err)
		require.NotNil(t, repos)
		assert.Equal(t, 3, len(repos))
	})

}
