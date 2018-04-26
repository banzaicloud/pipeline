package roles_test

import (
	"testing"

	"github.com/qor/roles"
)

func TestAllow(t *testing.T) {
	permission := roles.Allow(roles.Read, "api")

	if !permission.HasPermission(roles.Read, "api") {
		t.Errorf("API should has permission to Read")
	}

	if permission.HasPermission(roles.Update, "api") {
		t.Errorf("API should has no permission to Update")
	}

	if permission.HasPermission(roles.Read, "admin") {
		t.Errorf("admin should has no permission to Read")
	}

	if permission.HasPermission(roles.Update, "admin") {
		t.Errorf("admin should has no permission to Update")
	}
}

func TestDeny(t *testing.T) {
	permission := roles.Deny(roles.Create, "api")

	if !permission.HasPermission(roles.Read, "api") {
		t.Errorf("API should has permission to Read")
	}

	if !permission.HasPermission(roles.Update, "api") {
		t.Errorf("API should has permission to Update")
	}

	if permission.HasPermission(roles.Create, "api") {
		t.Errorf("API should has no permission to Update")
	}

	if !permission.HasPermission(roles.Read, "admin") {
		t.Errorf("admin should has permission to Read")
	}

	if !permission.HasPermission(roles.Create, "admin") {
		t.Errorf("admin should has permission to Update")
	}
}

func TestCRUD(t *testing.T) {
	permission := roles.Allow(roles.CRUD, "admin")
	if !permission.HasPermission(roles.Read, "admin") {
		t.Errorf("Admin should has permission to Read")
	}

	if !permission.HasPermission(roles.Update, "admin") {
		t.Errorf("Admin should has permission to Update")
	}

	if permission.HasPermission(roles.Read, "api") {
		t.Errorf("API should has no permission to Read")
	}

	if permission.HasPermission(roles.Update, "api") {
		t.Errorf("API should has no permission to Update")
	}
}

func TestAll(t *testing.T) {
	permission := roles.Allow(roles.Update, roles.Anyone)

	if permission.HasPermission(roles.Read, "api") {
		t.Errorf("API should has no permission to Read")
	}

	if !permission.HasPermission(roles.Update, "api") {
		t.Errorf("API should has permission to Update")
	}

	permission2 := roles.Deny(roles.Update, roles.Anyone)

	if !permission2.HasPermission(roles.Read, "api") {
		t.Errorf("API should has permission to Read")
	}

	if permission2.HasPermission(roles.Update, "api") {
		t.Errorf("API should has no permission to Update")
	}
}

func TestCustomizePermission(t *testing.T) {
	var customized roles.PermissionMode = "customized"
	permission := roles.Allow(customized, "admin")

	if !permission.HasPermission(customized, "admin") {
		t.Errorf("Admin should has customized permission")
	}

	if permission.HasPermission(roles.Read, "admin") {
		t.Errorf("Admin should has no permission to Read")
	}

	permission2 := roles.Deny(customized, "admin")

	if permission2.HasPermission(customized, "admin") {
		t.Errorf("Admin should has customized permission")
	}

	if !permission2.HasPermission(roles.Read, "admin") {
		t.Errorf("Admin should has no permission to Read")
	}
}
