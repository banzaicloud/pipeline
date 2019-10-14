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

package auth

import (
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite" // SQLite driver used for integration test
	"github.com/qor/auth"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:generate sh -c "which mockery > /dev/null && mockery -name OIDCOrganizationSyncer -inpkg -testonly || true"

func setUpDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open("sqlite3", "file::memory:")
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetOutput(ioutil.Discard)

	err = Migrate(db, logger)
	require.NoError(t, err)

	return db
}

func TestEmailToLoginName(t *testing.T) {
	loginName := emailToLoginName("john.doe@banzaicloud.com")

	if loginName != "john-doe-banzaicloud-com" {
		t.Error("loginName should be 'johndoe-banzaicloud' but is ", loginName)
	}
}

func TestBanzaiUserStorer_Update(t *testing.T) {
	t.Run("no change", func(t *testing.T) {
		db := setUpDatabase(t)
		orgSyncer := &MockOIDCOrganizationSyncer{}

		userStorer := BanzaiUserStorer{
			db:        db,
			orgSyncer: orgSyncer,
		}

		user1 := User{
			Name:  "John Doe",
			Email: "john.doe@example.com",
			Login: "john.doe",
			Organizations: []Organization{
				{
					Name:     "john.doe",
					Provider: "github",
				},
			},
		}

		user2 := User{
			Name:  "Jane Doe",
			Email: "jane.doe@example.com",
			Login: "jane.doe",
			Organizations: []Organization{
				{
					Name:     "jane.doe",
					Provider: "github",
				},
			},
		}

		user3 := User{
			Name:  "Ernest Doe",
			Email: "ernest.doe@example.com",
			Login: "ernest.doe",
			Organizations: []Organization{
				{
					Name:     "ernest.doe",
					Provider: "github",
				},
			},
		}

		var user2FromDB User

		err := db.Save(&user1).Error
		require.NoError(t, err)

		err = db.Save(&user2).Error
		require.NoError(t, err)

		err = db.Save(&user3).Error
		require.NoError(t, err)

		user2ID := "2"

		err = db.Where("id = ?", user2ID).First(&user2FromDB).Error
		require.NoError(t, err)

		assert.Equal(t, user2FromDB.Login, user2.Login, "user is expected to be user2")

		authCtx := auth.Context{Request: &http.Request{}}
		tokenClaims := IDTokenClaims{}

		orgSyncer.On("SyncOrganizations", authCtx.Request.Context(), user2FromDB, &tokenClaims).Return(nil)

		err = userStorer.Update(&auth.Schema{UID: user2ID, RawInfo: &tokenClaims}, &authCtx)
		require.NoError(t, err)
	})
}
