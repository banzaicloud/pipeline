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
	"encoding/json"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message/subscriber"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/auth"
)

func TestOrganizationSyncer_SyncOrganizations(t *testing.T) {
	db := setUpDatabase(t)
	store := NewGormOrganizationStore(db)
	publisher := gochannel.NewGoChannel(gochannel.Config{}, watermill.NopLogger{})
	const topic = "auth"
	eventBus, _ := cqrs.NewEventBus(publisher, func(_ string) string { return topic }, &cqrs.JSONMarshaler{})

	messages, err := publisher.Subscribe(context.Background(), topic)
	require.NoError(t, err)

	eventDispatcher := NewOrganizationEventDispatcher(eventBus)

	syncer := auth.NewOrganizationSyncer(store, eventDispatcher)

	user := auth.User{
		Name:  "John Doe",
		Email: "john.doe@example.com",
		Login: "john.doe",
	}

	err = db.Save(&user).Error
	require.NoError(t, err)

	currentMemberships := []auth.UserOrganization{
		{
			User: user,
			Organization: auth.Organization{
				Name:     "stays-the-same",
				Provider: "github",
			},
			Role: auth.RoleAdmin,
		},
		{
			User: user,
			Organization: auth.Organization{
				Name:     "change-role-to-member",
				Provider: "github",
			},
			Role: auth.RoleAdmin,
		},
		{
			User: user,
			Organization: auth.Organization{
				Name:     "change-role-to-admin",
				Provider: "github",
			},
			Role: auth.RoleMember,
		},
		{
			User:   user,
			UserID: user.ID,
			Organization: auth.Organization{
				Name:     "lose-access",
				Provider: "github",
			},
			OrganizationID: 4,
			Role:           auth.RoleAdmin,
		},
	}

	for _, currentMembership := range currentMemberships {
		err := db.Save(&currentMembership).Error
		require.NoError(t, err)
	}

	upstreamMemberships := []auth.UpstreamOrganizationMembership{
		{
			Organization: auth.UpstreamOrganization{
				Name:     "stays-the-same",
				Provider: "github",
			},
			Role: auth.RoleAdmin,
		},
		{
			Organization: auth.UpstreamOrganization{
				Name:     "change-role-to-member",
				Provider: "github",
			},
			Role: auth.RoleMember,
		},
		{
			Organization: auth.UpstreamOrganization{
				Name:     "change-role-to-admin",
				Provider: "github",
			},
			Role: auth.RoleAdmin,
		},
		{
			Organization: auth.UpstreamOrganization{
				Name:     "new-org",
				Provider: "github",
			},
			Role: auth.RoleAdmin,
		},
	}

	err = syncer.SyncOrganizations(context.Background(), user, upstreamMemberships)
	require.NoError(t, err)

	for _, m := range upstreamMemberships {
		var organization auth.Organization

		err := db.
			Where(auth.Organization{Name: m.Organization.Name}).
			First(&organization).
			Error
		require.NoError(t, err)

		var membership auth.UserOrganization

		err = db.
			Where(auth.UserOrganization{UserID: user.ID, OrganizationID: organization.ID}).
			First(&membership).
			Error
		require.NoError(t, err)

		assert.Equal(t, m.Role, membership.Role)

		if m.Organization.Name == "new-org" {
			received, all := subscriber.BulkRead(messages, 1, time.Second)
			if !all {
				t.Fatal("no message received")
			}

			var event auth.OrganizationCreated

			err := json.Unmarshal(received[0].Payload, &event)
			require.NoError(t, err)

			assert.Equal(
				t,
				auth.OrganizationCreated{
					ID:     organization.ID,
					UserID: user.ID,
				},
				event,
			)
		}
	}
}
