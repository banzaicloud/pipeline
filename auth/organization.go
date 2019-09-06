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
	"fmt"
	"time"

	"emperror.dev/errors"
)

// Organization represents a unit of users and resources.
type Organization struct {
	ID        uint      `gorm:"primary_key" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Name      string    `gorm:"unique;not null" json:"name"`
	Provider  string    `gorm:"not null" json:"provider"`
	Users     []User    `gorm:"many2many:user_organizations" json:"users,omitempty"`
	Role      string    `json:"-" gorm:"-"` // Used only internally
}

// IDString returns the ID as string.
func (org *Organization) IDString() string {
	return fmt.Sprint(org.ID)
}

// OrganizationSyncer synchronizes organization membership for a user.
// It creates missing organizations, adds user to and removes from existing organizations,
// updates organization role.
// Note: it never deletes organizations, only creates them if they are missing.
type OrganizationSyncer struct {
	store  OrganizationStore
	events OrganizationEvents
}

// NewOrganizationSyncer returns a new OrganizationSyncer.
func NewOrganizationSyncer(store OrganizationStore, events OrganizationEvents) OrganizationSyncer {
	return OrganizationSyncer{
		store:  store,
		events: events,
	}
}

// ErrOrganizationConflict is returned when an organization exists, but with mismatching parameters.
const ErrOrganizationConflict = errors.Sentinel("organization already exists, but with mismatching parameters")

// OrganizationStore is a persistence layer for organizations.
type OrganizationStore interface {
	// EnsureOrganizationExists ensures that an organization exists.
	// If one already exists with the same parameters it succeeds.
	// If one already exists with different parameters (eg. different provider),
	// it returns with an ErrOrganizationConflict error.
	// The function returns whether an organization was created or not, as well as it's ID.
	EnsureOrganizationExists(ctx context.Context, name string, provider string) (bool, uint, error)

	// GetOrganizationMembershipsOf returns the list of organization memberships for a user.
	GetOrganizationMembershipsOf(ctx context.Context, userID uint) ([]UserOrganization, error)

	// RemoveUserFromOrganization removes a user from an organization.
	RemoveUserFromOrganization(ctx context.Context, organizationID uint, userID uint) error

	// ApplyUserMembership ensures that a user is a member of an organization with the necessary role.
	ApplyUserMembership(ctx context.Context, organizationID uint, userID uint, role string) error
}

// OrganizationEvents dispatches organization events.
type OrganizationEvents interface {
	// OrganizationCreated dispatches an OrganizationCreated event.
	OrganizationCreated(ctx context.Context, event OrganizationCreated) error
}

// OrganizationCreated event is triggered when an organization is created in the system.
type OrganizationCreated struct {
	// ID is the created organization ID.
	ID uint

	// UserID is the ID of the user whose login triggered the organization being created.
	UserID uint
}

// UpstreamOrganizationMembership represents an organization membership of a user
// from the upstream authentication source.
type UpstreamOrganizationMembership struct {
	Organization UpstreamOrganization
	Role         string
}

// UpstreamOrganization represents an organization from the upstream authentication source.
type UpstreamOrganization struct {
	Name     string
	Provider string
}

// SyncOrganizations synchronizes organization membership for a user.
func (s OrganizationSyncer) SyncOrganizations(ctx context.Context, user User, upstreamMemberships []UpstreamOrganizationMembership) error {
	membershipsToAdd := make(map[string]string, len(upstreamMemberships))
	organizationsCreated := make(map[string]uint)

	for _, membership := range upstreamMemberships {
		created, id, err := s.store.EnsureOrganizationExists(
			ctx,
			membership.Organization.Name,
			membership.Organization.Provider,
		)
		if err != nil {
			return errors.WithDetails(err, "userId", user.ID)
		}

		membershipsToAdd[membership.Organization.Name] = membership.Role

		if created {
			organizationsCreated[membership.Organization.Name] = id

			event := OrganizationCreated{
				ID:     id,
				UserID: user.ID,
			}

			err := s.events.OrganizationCreated(ctx, event)
			if err != nil {
				return err
			}
		}
	}

	currentMemberships, err := s.store.GetOrganizationMembershipsOf(ctx, user.ID)
	if err != nil {
		return err
	}

	for _, currentMembership := range currentMemberships {
		role, ok := membershipsToAdd[currentMembership.Organization.Name]

		// User is not in the list of upstream memberships, remove from organization
		if !ok {
			err := s.store.RemoveUserFromOrganization(ctx, currentMembership.OrganizationID, user.ID)
			if err != nil {
				return err
			}

			continue
		}

		// Membership is already up to date, there is nothing to do
		if currentMembership.Role == role {
			// Membership already exists, no need to add
			delete(membershipsToAdd, currentMembership.Organization.Name)

			continue
		}

		err := s.store.ApplyUserMembership(ctx, currentMembership.OrganizationID, user.ID, role)
		if err != nil {
			return err
		}

		// Membership already exists, no need to add
		delete(membershipsToAdd, currentMembership.Organization.Name)
	}

	for organizationName, role := range membershipsToAdd {
		err := s.store.ApplyUserMembership(ctx, organizationsCreated[organizationName], user.ID, role)
		if err != nil {
			return err
		}
	}

	return nil
}
