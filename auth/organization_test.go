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
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:generate sh -c "test -x ${MOCKERY} && ${MOCKERY} -name OrganizationStore -inpkg -testonly"
//go:generate sh -c "test -x ${MOCKERY} && ${MOCKERY} -name OrganizationEvents -inpkg -testonly"

func TestOrganizationSyncer_SyncOrganizations(t *testing.T) { // TODO: rewrite this test with an in-memory store
	store := &MockOrganizationStore{}
	events := &MockOrganizationEvents{}
	syncer := NewOrganizationSyncer(store, events)

	ctx := context.Background()

	user := User{
		ID: 1,
	}

	upstreamMemberships := []UpstreamOrganizationMembership{
		{
			Organization: UpstreamOrganization{
				Name:     "stays-the-same",
				Provider: "github",
			},
			Role: RoleAdmin,
		},
		{
			Organization: UpstreamOrganization{
				Name:     "change-role-to-member",
				Provider: "github",
			},
			Role: RoleMember,
		},
		{
			Organization: UpstreamOrganization{
				Name:     "change-role-to-admin",
				Provider: "github",
			},
			Role: RoleAdmin,
		},
		{
			Organization: UpstreamOrganization{
				Name:     "new-org",
				Provider: "github",
			},
			Role: RoleAdmin,
		},
	}

	for _, upstreamMembership := range upstreamMemberships {
		store.On(
			"EnsureOrganizationExists",
			ctx,
			upstreamMembership.Organization.Name,
			upstreamMembership.Organization.Provider,
		).Return(true, uint(5), nil)

		events.On("OrganizationCreated", ctx, OrganizationCreated{5, user.ID}).Return(nil)
	}

	currentMemberships := []UserOrganization{
		{
			User:   user,
			UserID: user.ID,
			Organization: Organization{
				ID:       1,
				Name:     "stays-the-same",
				Provider: "github",
			},
			OrganizationID: 1,
			Role:           RoleAdmin,
		},
		{
			User:   user,
			UserID: user.ID,
			Organization: Organization{
				ID:       2,
				Name:     "change-role-to-member",
				Provider: "github",
			},
			OrganizationID: 2,
			Role:           RoleAdmin,
		},
		{
			User:   user,
			UserID: user.ID,
			Organization: Organization{
				ID:       3,
				Name:     "change-role-to-admin",
				Provider: "github",
			},
			OrganizationID: 3,
			Role:           RoleMember,
		},
		{
			User:   user,
			UserID: user.ID,
			Organization: Organization{
				ID:       4,
				Name:     "lose-access",
				Provider: "github",
			},
			OrganizationID: 4,
			Role:           RoleAdmin,
		},
	}

	store.On("GetOrganizationMembershipsOf", ctx, user.ID).Return(currentMemberships, nil)
	store.On("RemoveUserFromOrganization", ctx, currentMemberships[3].OrganizationID, user.ID).Return(nil)
	store.On("ApplyUserMembership", ctx, currentMemberships[1].OrganizationID, user.ID, RoleMember).Return(nil)
	store.On("ApplyUserMembership", ctx, currentMemberships[2].OrganizationID, user.ID, RoleAdmin).Return(nil)
	store.On("ApplyUserMembership", ctx, uint(5), user.ID, RoleAdmin).Return(nil)

	err := syncer.SyncOrganizations(ctx, user, upstreamMemberships)
	require.NoError(t, err)

	store.AssertExpectations(t)
	events.AssertExpectations(t)
}
