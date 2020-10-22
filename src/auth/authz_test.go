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

package auth

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/common"
)

func TestRbacEnforcer_Enforce_NoOrgIsAllowed(t *testing.T) {
	enforcer := NewRbacEnforcer(nil, NewServiceAccountService(), common.NoopLogger{})

	ok, err := enforcer.Enforce(nil, &User{}, "/", "GET", nil)
	require.NoError(t, err)

	assert.True(t, ok)
}

func TestRbacEnforcer_Enforce_NoUserIsNotAllowed(t *testing.T) {
	enforcer := NewRbacEnforcer(nil, NewServiceAccountService(), common.NoopLogger{})

	ok, err := enforcer.Enforce(&Organization{}, nil, "/", "GET", nil)
	require.NoError(t, err)

	assert.False(t, ok)
}

func TestRbacEnforcer_Enforce_VirtualUser(t *testing.T) {
	org := Organization{
		ID:   1,
		Name: "example",
	}

	tests := []struct {
		organization Organization
		user         User
		path         string
		query        url.Values
	}{
		{
			organization: org,
			user: User{
				ID:    0,
				Login: "clusters/1/2",
			},
			path: "/api/v1/orgs/1/clusters/2/pke/status",
		},
		{
			organization: org,
			user: User{
				ID:    0,
				Login: "clusters/1/2",
			},
			path: "/api/v1/orgs/1/clusters/2/pke/ready",
		},
		{
			organization: org,
			user: User{
				ID:    0,
				Login: "clusters/1/2",
			},
			path: "/api/v1/orgs/1/clusters/2/pke/leader",
		},
		{
			organization: org,
			user: User{
				ID:    0,
				Login: "clusters/1/2",
			},
			path: "/api/v1/orgs/1/clusters/2/bootstrap",
		},
		{
			organization: org,
			user: User{
				ID:    0,
				Login: "clusters/1/2",
			},
			path: "/api/v1/orgs/1/secrets",
			query: url.Values{
				"tags": []string{"clusterID:2"},
				"type": []string{"pkecert"},
			},
		},
		{
			organization: org,
			user: User{
				ID:    0,
				Login: "example",
			},
		},
		{
			organization: org,
			user: User{
				ID:             0,
				Login:          "pipeline",
				ServiceAccount: true,
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run("", func(t *testing.T) {
			enforcer := NewRbacEnforcer(nil, NewServiceAccountService(), common.NoopLogger{})

			ok, err := enforcer.Enforce(&test.organization, &test.user, test.path, "GET", test.query)
			require.NoError(t, err)

			assert.True(t, ok)
		})
	}
}

func TestRbacEnforcer_Enforce_VirtualUser_Invalid(t *testing.T) {
	org := Organization{
		ID:   1,
		Name: "example",
	}

	tests := []struct {
		organization Organization
		user         User
		path         string
		error        bool
		query        url.Values
	}{
		{
			organization: org,
			user: User{
				ID:    0,
				Login: "clusters/1",
			},
			error: false,
		},
		{
			organization: org,
			user: User{
				ID:    0,
				Login: "clusters/1/2",
			},
			path:  "/api/v1/orgs/1/clusters/3/pke/status",
			error: false,
		},
		{
			organization: org,
			user: User{
				ID:    0,
				Login: "clusters/1/2",
			},
			path:  "/api/v1/orgs/1/clusters/2/pke/somethingwild",
			error: false,
		},
		{
			organization: org,
			user: User{
				ID:    0,
				Login: "clusters/1/2",
			},
			path:  "/api/v1/orgs/1/clusters/2",
			error: false,
		},
		{
			organization: org,
			user: User{
				ID:    0,
				Login: "clusters/1/2",
			},
			path:  "/api/v1/orgs/1/clusters/2/config",
			error: false,
		},
		{
			organization: org,
			user: User{
				ID:    0,
				Login: "clusters/1/2",
			},
			path:  "/api/v1/orgs/1",
			error: false,
		},
		{
			organization: org,
			user: User{
				ID:    0,
				Login: "clusters/1/2",
			},
			path:  "/api/v1/orgs/1/secrets",
			error: false,
		},
		{
			organization: org,
			user: User{
				ID:    0,
				Login: "clusters/1/2",
			},
			path: "/api/v1/orgs/1/secrets",
			query: url.Values{
				"tags": []string{"clusterID:2"},
			},
			error: false,
		},
		{
			organization: org,
			user: User{
				ID:    0,
				Login: "clusters/1/2",
			},
			path: "/api/v1/orgs/1/secrets",
			query: url.Values{
				"tags": []string{"clusterID:2"},
				"type": []string{"config"},
			},
			error: false,
		},
		{
			organization: org,
			user: User{
				ID:    0,
				Login: "clusters/1/2",
			},
			path: "/api/v1/orgs/1/secrets",
			query: url.Values{
				"tags:": []string{"clusterID:3"},
				"type":  []string{"pkecert"},
			},
			error: false,
		},
		{
			organization: org,
			user: User{
				ID:    0,
				Login: "clusters/example/2",
			},
			error: true,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.user.Login, func(t *testing.T) {
			enforcer := NewRbacEnforcer(nil, NewServiceAccountService(), common.NoopLogger{})

			ok, err := enforcer.Enforce(&test.organization, &test.user, test.path, "GET", test.query)
			if test.error {
				require.Error(t, err)
			}

			assert.False(t, ok)
		})
	}
}

func TestRbacEnforcer_Enforce_NotAMember(t *testing.T) {
	org := Organization{
		ID:   1,
		Name: "example",
	}

	user := User{
		ID:    1,
		Login: "john.doe",
	}

	roleSource := &MockRoleSource{}
	roleSource.On("FindUserRole", mock.Anything, org.ID, user.ID).Return("", false, nil)

	enforcer := NewRbacEnforcer(roleSource, NewServiceAccountService(), common.NoopLogger{})

	ok, err := enforcer.Enforce(&org, &user, "/", "GET", nil)
	require.NoError(t, err)

	assert.False(t, ok)
}

func TestRbacEnforcer_Enforce(t *testing.T) {
	org := Organization{
		ID:   1,
		Name: "example",
	}

	user := User{
		ID:    1,
		Login: "john.doe",
	}

	tests := []struct {
		role     string
		path     string
		method   string
		expected bool
	}{
		{
			role:     RoleAdmin,
			path:     "/",
			method:   "GET",
			expected: true,
		},
		{
			role:     RoleAdmin,
			path:     "/",
			method:   "POST",
			expected: true,
		},
		{
			role:     RoleAdmin,
			path:     "/api/v1/orgs/1/secrets/secretID",
			method:   "GET",
			expected: true,
		},
		{
			role:     RoleAdmin,
			path:     "/api/v1/orgs/1/secrets",
			method:   "POST",
			expected: true,
		},
		{
			role:     RoleMember,
			path:     "/",
			method:   "GET",
			expected: true,
		},
		{
			role:     RoleMember,
			path:     "/",
			method:   "POST",
			expected: true,
		},
		{
			role:     RoleMember,
			path:     "/api/v1/orgs/1/buckets",
			method:   "HEAD",
			expected: true,
		},
		{
			role:     RoleMember,
			path:     "/api/v1/orgs/1/clusters",
			method:   "POST",
			expected: false,
		},
		{
			role:     RoleMember,
			path:     "/api/v1/orgs/1/secrets/secretID",
			method:   "GET",
			expected: false,
		},
		{
			role:     RoleMember,
			path:     "/api/v1/orgs/1/secrets",
			method:   "POST",
			expected: false,
		},
		{
			role:     RoleMember,
			path:     "/api/v1/orgs/1/clusters/1/config",
			method:   "GET",
			expected: false,
		},
	}

	for _, test := range tests {
		test := test

		t.Run("", func(t *testing.T) {
			roleSource := &MockRoleSource{}
			roleSource.On("FindUserRole", mock.Anything, org.ID, user.ID).Return(test.role, true, nil)

			enforcer := NewRbacEnforcer(roleSource, NewServiceAccountService(), common.NoopLogger{})

			ok, err := enforcer.Enforce(&org, &user, test.path, test.method, nil)
			require.NoError(t, err)

			assert.Equal(t, test.expected, ok)
		})
	}
}
