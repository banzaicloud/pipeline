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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
)

//go:generate sh -c "which mockery > /dev/null && mockery -name RoleSource -inpkg -testonly || true"

func TestRbacEnforcer_Enforce_NoOrgIsAllowed(t *testing.T) {
	enforcer := NewRbacEnforcer(nil, commonadapter.NewNoopLogger())

	ok, err := enforcer.Enforce(nil, &User{}, "/", "GET")
	require.NoError(t, err)

	assert.True(t, ok)
}

func TestRbacEnforcer_Enforce_NoUserIsNotAllowed(t *testing.T) {
	enforcer := NewRbacEnforcer(nil, commonadapter.NewNoopLogger())

	ok, err := enforcer.Enforce(&Organization{}, nil, "/", "GET")
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
		expected     bool
	}{
		{
			organization: org,
			user: User{
				ID:    0,
				Login: "clusters/1",
			},
			expected: true,
		},
		{
			organization: org,
			user: User{
				ID:    0,
				Login: "example",
			},
			expected: true,
		},
	}

	for _, test := range tests {
		test := test

		t.Run("", func(t *testing.T) {
			enforcer := NewRbacEnforcer(nil, commonadapter.NewNoopLogger())

			ok, err := enforcer.Enforce(&test.organization, &test.user, "/", "GET")
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
		error        bool
	}{
		{
			organization: org,
			user: User{
				ID:    0,
				Login: "clusters/",
			},
			error: false,
		},
		{
			organization: org,
			user: User{
				ID:    0,
				Login: "clusters/example",
			},
			error: true,
		},
	}

	for _, test := range tests {
		test := test

		t.Run("", func(t *testing.T) {
			enforcer := NewRbacEnforcer(nil, commonadapter.NewNoopLogger())

			ok, err := enforcer.Enforce(&test.organization, &test.user, "/", "GET")
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

	enforcer := NewRbacEnforcer(roleSource, commonadapter.NewNoopLogger())

	ok, err := enforcer.Enforce(&org, &user, "/", "GET")
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
	}

	for _, test := range tests {
		test := test

		t.Run("", func(t *testing.T) {
			roleSource := &MockRoleSource{}
			roleSource.On("FindUserRole", mock.Anything, org.ID, user.ID).Return(test.role, true, nil)

			enforcer := NewRbacEnforcer(roleSource, commonadapter.NewNoopLogger())

			ok, err := enforcer.Enforce(&org, &user, test.path, test.method)
			require.NoError(t, err)

			assert.Equal(t, test.expected, ok)
		})
	}
}
