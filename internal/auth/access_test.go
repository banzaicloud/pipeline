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

	gormadapter "github.com/casbin/gorm-adapter"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/stretchr/testify/assert"
)

func TestAccessManager_DefaultPolicies(t *testing.T) {
	adapter := gormadapter.NewAdapter("sqlite3", "file::memory:")
	enforcer := NewEnforcer(adapter)
	accessManager := NewAccessManager(enforcer, "")

	enforcer.ClearPolicy()

	accessManager.AddDefaultPolicies()

	accessManager.GrantDefaultAccessToUser("user")
	accessManager.GrantDefaultAccessToVirtualUser("userVirtual")

	tests := []struct {
		user           string
		path           string
		method         string
		expectedResult bool
	}{
		{
			user:           "user",
			path:           "/api/v1/allowed/secrets",
			method:         http.MethodGet,
			expectedResult: true,
		},
		{
			user:           "user",
			path:           "/api/v1/allowed/secrets/asd",
			method:         http.MethodGet,
			expectedResult: true,
		},
		{
			user:           "user",
			path:           "/api/v1/orgs",
			method:         http.MethodGet,
			expectedResult: true,
		},
		{
			user:           "user",
			path:           "/api/v1/token",
			method:         http.MethodGet,
			expectedResult: true,
		},
		{
			user:           "user",
			path:           "/api/v1/tokens",
			method:         http.MethodGet,
			expectedResult: true,
		},
		{
			user:           "user",
			path:           "/api/v1/orgs/1",
			method:         http.MethodGet,
			expectedResult: false,
		},
		{
			user:           "userVirtual",
			path:           "/api/v1/orgs",
			method:         http.MethodGet,
			expectedResult: true,
		},
		{
			user:           "userVirtual",
			path:           "/api/v1/allowed/secrets",
			method:         http.MethodGet,
			expectedResult: false,
		},
	}

	for _, test := range tests {
		test := test

		t.Run("", func(t *testing.T) {
			granted := enforcer.Enforce(test.user, test.path, test.method)

			assert.Equal(t, test.expectedResult, granted)
		})
	}
}

func TestAccessManager_OrganizationPolicies(t *testing.T) {
	adapter := gormadapter.NewAdapter("sqlite3", "file::memory:")
	enforcer := NewEnforcer(adapter)
	accessManager := NewAccessManager(enforcer, "")

	enforcer.ClearPolicy()

	accessManager.AddOrganizationPolicies(1)
	accessManager.GrantOrganizationAccessToUser("user", 1)

	// Granting the same access twice should be idempotent
	accessManager.AddOrganizationPolicies(1)
	accessManager.GrantOrganizationAccessToUser("user", 1)

	tests := []struct {
		path           string
		method         string
		expectedResult bool
	}{
		{
			path:           "/api/v1/orgs/1",
			method:         http.MethodGet,
			expectedResult: true,
		},
		{
			path:           "/api/v1/orgs/1/clusters",
			method:         http.MethodGet,
			expectedResult: true,
		},
		{
			path:           "/dashboard/orgs/1/clusters",
			method:         http.MethodGet,
			expectedResult: true,
		},
		{
			path:           "/api/v1/orgs/2",
			method:         http.MethodGet,
			expectedResult: false,
		},
	}

	for _, test := range tests {
		test := test

		t.Run("", func(t *testing.T) {
			granted := enforcer.Enforce("user", test.path, test.method)

			assert.Equal(t, test.expectedResult, granted)
		})
	}
}

func TestAccessManager_RevokeOrganizationAccessFromUser(t *testing.T) {
	adapter := gormadapter.NewAdapter("sqlite3", "file::memory:")
	enforcer := NewEnforcer(adapter)
	accessManager := NewAccessManager(enforcer, "")

	enforcer.ClearPolicy()

	accessManager.AddOrganizationPolicies(1)
	accessManager.AddOrganizationPolicies(2)

	accessManager.GrantOrganizationAccessToUser("user", 1)
	accessManager.GrantOrganizationAccessToUser("user", 2)
	accessManager.RevokeOrganizationAccessFromUser("user", 1)

	tests := []struct {
		path           string
		method         string
		expectedResult bool
	}{
		{
			path:           "/api/v1/orgs/1",
			method:         http.MethodGet,
			expectedResult: false,
		},
		{
			path:           "/api/v1/orgs/1/clusters",
			method:         http.MethodGet,
			expectedResult: false,
		},
		{
			path:           "/dashboard/orgs/1/clusters",
			method:         http.MethodGet,
			expectedResult: false,
		},
		{
			path:           "/api/v1/orgs/2",
			method:         http.MethodGet,
			expectedResult: true,
		},
		{
			path:           "/api/v1/orgs/2/clusters",
			method:         http.MethodGet,
			expectedResult: true,
		},
		{
			path:           "/dashboard/orgs/2/clusters",
			method:         http.MethodGet,
			expectedResult: true,
		},
	}

	for _, test := range tests {
		test := test

		t.Run("", func(t *testing.T) {
			granted := enforcer.Enforce("user", test.path, test.method)

			assert.Equal(t, test.expectedResult, granted)
		})
	}
}

func TestAccessManager_RevokeAllAccessFromUser(t *testing.T) {
	adapter := gormadapter.NewAdapter("sqlite3", "file::memory:")
	enforcer := NewEnforcer(adapter)
	accessManager := NewAccessManager(enforcer, "")

	enforcer.ClearPolicy()

	accessManager.AddDefaultPolicies()
	accessManager.AddOrganizationPolicies(1)

	accessManager.GrantOrganizationAccessToUser("user", 1)

	accessManager.RevokeAllAccessFromUser("user")

	tests := []struct {
		path           string
		method         string
		expectedResult bool
	}{
		{
			path:           "/api/v1/allowed/secrets",
			method:         http.MethodGet,
			expectedResult: false,
		},
		{
			path:           "/api/v1/allowed/secrets/asd",
			method:         http.MethodGet,
			expectedResult: false,
		},
		{
			path:           "/api/v1/orgs",
			method:         http.MethodGet,
			expectedResult: false,
		},
		{
			path:           "/api/v1/token",
			method:         http.MethodGet,
			expectedResult: false,
		},
		{
			path:           "/api/v1/tokens",
			method:         http.MethodGet,
			expectedResult: false,
		},
		{
			path:           "/api/v1/orgs/1",
			method:         http.MethodGet,
			expectedResult: false,
		},
		{
			path:           "/api/v1/orgs/1/clusters",
			method:         http.MethodGet,
			expectedResult: false,
		},
		{
			path:           "/dashboard/orgs/1/clusters",
			method:         http.MethodGet,
			expectedResult: false,
		},
	}

	for _, test := range tests {
		test := test

		t.Run("", func(t *testing.T) {
			granted := enforcer.Enforce("user", test.path, test.method)

			assert.Equal(t, test.expectedResult, granted)
		})
	}
}
