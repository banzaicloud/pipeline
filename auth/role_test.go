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
	rawBindings := map[string]string{
		RoleAdmin:  "admin",
		RoleMember: "member",
	}

	roleBinder, err := NewRoleBinder(rawBindings)
	if err != nil {
		t.Fatal(err)
	}

	groups := []string{"none", "member", "admin"}

	role := roleBinder.BindRole(groups)

	if role != RoleAdmin {
		t.Errorf("the expected role is %q, received: %q", RoleAdmin, role)
	}
}
