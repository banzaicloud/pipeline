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
	"net/http"
	"testing"

	"github.com/banzaicloud/pipeline/auth"
	pkgAuth "github.com/banzaicloud/pipeline/pkg/auth"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/stretchr/testify/assert"
)

func newOrg(t *testing.T, db *gorm.DB, id uint, name string) *auth.Organization {
	org := auth.Organization{ID: pkgAuth.OrganizationID(id), Name: name}
	db.AutoMigrate(org)
	if err := db.FirstOrCreate(&org).Error; err != nil {
		t.Fatal(err)
	}
	return &org
}

func newUser(t *testing.T, db *gorm.DB, id uint, login string) *auth.User {
	user := auth.User{ID: pkgAuth.UserID(id), Login: login}
	db.AutoMigrate(user)
	if err := db.FirstOrCreate(&user).Error; err != nil {
		t.Fatal(err)
	}
	return &user
}

func addUserToOrg(t *testing.T, db *gorm.DB, user *auth.User, org *auth.Organization) {
	db.AutoMigrate(auth.UserOrganization{})
	if err := db.FirstOrCreate(&auth.UserOrganization{OrganizationID: org.ID, UserID: user.ID}).Error; err != nil {
		t.Fatal(err)
	}
}

func TestAccessManager_DefaultPolicies(t *testing.T) {
	db, err := gorm.Open("sqlite3", "file::memory:")
	if err != nil {
		t.Fatal(err)
	}
	enforcer := NewEnforcer(db)
	accessManager := NewAccessManager(enforcer, "")

	accessManager.AddDefaultPolicies()

	accessManager.GrantDefaultAccessToUser("user")
	accessManager.GrantDefaultAccessToVirtualUser("userVirtual")

	user1 := newUser(t, db, 1, "user1")
	user2 := newUser(t, db, 2, "user2")

	org1 := newOrg(t, db, 1, "user1")
	org2 := newOrg(t, db, 1, "user2")

	addUserToOrg(t, db, user2, org2)

	tests := []struct {
		org            *auth.Organization
		user           *auth.User
		path           string
		method         string
		expectedResult bool
	}{
		{
			org:            nil,
			user:           user1,
			path:           "/api/v1/allowed/secrets",
			method:         http.MethodGet,
			expectedResult: true,
		},
		{
			org:            nil,
			user:           user1,
			path:           "/api/v1/allowed/secrets/asd",
			method:         http.MethodGet,
			expectedResult: true,
		},
		{
			org:            nil,
			user:           user1,
			path:           "/api/v1/orgs",
			method:         http.MethodGet,
			expectedResult: true,
		},
		{
			org:            nil,
			user:           user1,
			path:           "/api/v1/token",
			method:         http.MethodGet,
			expectedResult: true,
		},
		{
			org:            nil,
			user:           user1,
			path:           "/api/v1/tokens",
			method:         http.MethodGet,
			expectedResult: true,
		},
		{
			org:            org1,
			user:           user1,
			path:           "/api/v1/orgs/1",
			method:         http.MethodGet,
			expectedResult: false,
		},
		{
			org:            org2,
			user:           user2,
			path:           "/api/v1/orgs/1",
			method:         http.MethodGet,
			expectedResult: true,
		},
		{
			user:           user2,
			path:           "/api/v1/orgs",
			method:         http.MethodGet,
			expectedResult: true,
		},
		{
			user:           user2,
			path:           "/api/v1/allowed/secrets",
			method:         http.MethodGet,
			expectedResult: true,
		},
	}

	for _, test := range tests {
		test := test

		t.Run("", func(t *testing.T) {
			granted, err := enforcer.Enforce(test.org, test.user, test.path, test.method)
			if err != nil {
				t.Fatal(err.Error())
			}

			assert.Equal(t, test.expectedResult, granted)
		})
	}
}
