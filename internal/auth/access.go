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
	pkgAuth "github.com/banzaicloud/pipeline/pkg/auth"
)

// AccessManager is responsible for managing authorization rules.
// NOTE:
// Currently this implementation is a dummy one, and is a placeholder for a future implementation.
// It hasn't been removed to mark the places where it should been called.
type AccessManager struct {
	enforcer Enforcer
	basePath string
}

// NewAccessManager returns a new AccessManager instance.
func NewAccessManager(enforcer Enforcer, basePath string) *AccessManager {
	return &AccessManager{
		enforcer: enforcer,
		basePath: basePath,
	}
}

// AddDefaultPolicies adds default policy rules to the underlying access manager.
func (m *AccessManager) AddDefaultPolicies() {
}

// GrantDefaultAccessToUser adds all the default non-org-specific role to a user.
func (m *AccessManager) GrantDefaultAccessToUser(userID string) {
}

// GrantDefaultAccessToVirtualUser adds org list role to a virtual user.
func (m *AccessManager) GrantDefaultAccessToVirtualUser(userID string) {
}

// AddOrganizationPolicies creates an organization role, by adding the default (*) org policies for the given organization.
func (m *AccessManager) AddOrganizationPolicies(orgID pkgAuth.OrganizationID) {
}

// GrantOrganizationAccessToUser adds a user to an organization by adding the associated organization role.
func (m *AccessManager) GrantOrganizationAccessToUser(userID string, orgID pkgAuth.OrganizationID) {
}

// RevokeOrganizationAccessFromUser removes a user from an organization by removing the associated organization role.
func (m *AccessManager) RevokeOrganizationAccessFromUser(userID string, orgID pkgAuth.OrganizationID) {
}

// RevokeAllAccessFromUser removes all roles for a given user.
func (m *AccessManager) RevokeAllAccessFromUser(userID string) {
}
