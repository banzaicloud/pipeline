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
)

func TestRoleBinder_BindRole(t *testing.T) {
	tests := []struct {
		defaultRole string
		rawBindings map[string]string
		groups      []string
		role        string
	}{
		{
			defaultRole: RoleAdmin,
			rawBindings: map[string]string{
				RoleAdmin:  "admin",
				RoleMember: "member",
			},
			groups: []string{"none", "member", "admin"},
			role:   RoleAdmin,
		},
		{
			defaultRole: RoleAdmin,
			rawBindings: map[string]string{
				RoleAdmin:  "admin",
				RoleMember: "member",
			},
			groups: []string{"none"},
			role:   RoleAdmin,
		},
		{
			defaultRole: RoleAdmin,
			rawBindings: map[string]string{
				RoleAdmin:  "admin",
				RoleMember: "member",
			},
			groups: []string{"admin"},
			role:   RoleAdmin,
		},
		{
			defaultRole: RoleAdmin,
			rawBindings: map[string]string{
				RoleAdmin:  ".*",
				RoleMember: "",
			},
			groups: []string{"admin"},
			role:   RoleAdmin,
		},
		{
			defaultRole: RoleAdmin,
			rawBindings: map[string]string{
				RoleAdmin:  ".*",
				RoleMember: "",
			},
			groups: []string{},
			role:   RoleAdmin,
		},
		{
			defaultRole: "",
			rawBindings: map[string]string{
				RoleAdmin:  ".*",
				RoleMember: "",
			},
			groups: []string{},
			role:   RoleMember,
		},
	}

	t.Parallel()

	for _, test := range tests {
		test := test

		t.Run("", func(t *testing.T) {
			roleBinder, err := NewRoleBinder(test.defaultRole, test.rawBindings)
			if err != nil {
				t.Fatal(err)
			}

			role := roleBinder.BindRole(test.groups)

			if role != test.role {
				t.Errorf("the expected role is %q, received: %q", test.role, role)
			}
		})
	}
}

func TestRoleBinder_NewRoleBinder_InvalidRole(t *testing.T) {
	rawBindings := map[string]string{
		"totally_invalid": "",
	}

	_, err := NewRoleBinder(RoleAdmin, rawBindings)
	if err == nil {
		t.Error("invalid role should result in an error")
	}
}

func TestRoleBinder_NewRoleBinder_InvalidRegex(t *testing.T) {
	rawBindings := map[string]string{
		RoleAdmin: "[",
	}

	_, err := NewRoleBinder(RoleAdmin, rawBindings)
	if err == nil {
		t.Error("invalid regex should result in an error")
	}
}
