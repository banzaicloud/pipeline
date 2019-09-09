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
	"regexp"

	"emperror.dev/errors"
)

const (
	RoleAdmin  = "admin"
	RoleMember = "member"
)

// nolint: gochecknoglobals
// The order of roles goes from lowest to highest.
var roles = []string{
	RoleMember,
	RoleAdmin,
}

// nolint: gochecknoinits
func init() {
	roleIndex = make(map[string]int, len(roles))
	for i, role := range roles {
		roleIndex[role] = i + 1
	}
}

// nolint: gochecknoglobals
var roleIndex map[string]int

// nolint: gochecknoglobals
var roleLevelMap = map[string]int{
	RoleAdmin:  100,
	RoleMember: 50,
}

// RoleBinder binds groups from an OIDC ID token to Pipeline roles.
type RoleBinder struct {
	bindings map[string]*regexp.Regexp
}

// NewRoleBinder returns a new RoleBinder.
func NewRoleBinder(rawBindings map[string]string) (RoleBinder, error) {
	rb := RoleBinder{
		bindings: make(map[string]*regexp.Regexp, len(rawBindings)),
	}

	for role, rule := range rawBindings {
		if _, ok := roleIndex[role]; !ok {
			return rb, errors.NewWithDetails("invalid role", "role", role)
		}

		r, err := regexp.Compile(rule)
		if err != nil {
			return rb, errors.NewWithDetails("invalid role binding rule", "rule", rule)
		}

		rb.bindings[role] = r
	}

	return rb, nil
}

// BindRole binds the highest possible role to the list of provided groups.
func (rb RoleBinder) BindRole(groups []string) string {
	// Assign the lowest role to the user by default.
	currentRole := RoleMember

	for _, group := range groups {
		for role, rule := range rb.bindings {
			if rule.MatchString(group) && roleIndex[currentRole] < roleIndex[role] {
				currentRole = role
			}
		}
	}

	return currentRole
}
