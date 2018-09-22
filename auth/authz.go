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
	"fmt"
	"net/http"
	"time"

	"github.com/casbin/casbin"
	"github.com/casbin/gorm-adapter"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

const modelDefinition = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch(r.obj, p.obj) && (r.act == p.act || p.act == "*")
`

const logging = false

var enforcer *casbin.SyncedEnforcer

// NewAuthorizer returns the MySQL based default authorizer
func NewAuthorizer(dsn string) gin.HandlerFunc {
	adapter := gormadapter.NewAdapter("mysql", dsn, true)
	model := casbin.NewModel(modelDefinition)
	enforcer = casbin.NewSyncedEnforcer(model, adapter, logging)
	enforcer.StartAutoLoadPolicy(10 * time.Second)
	addDefaultPolicies()
	return newAuthorizer(enforcer)
}

// NewAuthorizer returns the authorizer, uses a Casbin enforcer as input
func newAuthorizer(e *casbin.SyncedEnforcer) gin.HandlerFunc {
	return func(c *gin.Context) {
		a := &BearerAuthorizer{enforcer: e}

		if !a.CheckPermission(c.Request) {
			a.RequirePermission(c)
		}
	}
}

// BearerAuthorizer stores the casbin handler
type BearerAuthorizer struct {
	enforcer *casbin.SyncedEnforcer
}

// GetUserID gets the user name from the request.
// Currently, only HTTP Bearer token authentication is supported
func (a *BearerAuthorizer) GetUserID(r *http.Request) string {
	user := GetCurrentUser(r)
	if user.ID == 0 {
		return user.Login // This is needed for Drone virtual user tokens
	}
	return user.IDString()
}

// CheckPermission checks the user/method/path combination from the request.
// Returns true (permission granted) or false (permission forbidden)
func (a *BearerAuthorizer) CheckPermission(r *http.Request) bool {
	userID := a.GetUserID(r)
	method := r.Method
	path := r.URL.Path
	return a.enforcer.Enforce(userID, path, method)
}

// RequirePermission returns the 403 Forbidden to the client
func (a *BearerAuthorizer) RequirePermission(c *gin.Context) {
	c.AbortWithStatus(http.StatusForbidden)
}

func addDefaultPolicies() {
	basePath := viper.GetString("pipeline.basepath")
	enforcer.AddPolicy("default", basePath+"/api/v1/allowed/secrets", "*")
	enforcer.AddPolicy("default", basePath+"/api/v1/allowed/secrets/*", "*")
	enforcer.AddPolicy("default", basePath+"/api/v1/orgs", "*")
	enforcer.AddPolicy("default", basePath+"/api/v1/token", "*")
	enforcer.AddPolicy("default", basePath+"/api/v1/tokens", "*")
	enforcer.AddPolicy("defaultVirtual", basePath+"/api/v1/orgs", "GET")
}

// AddDefaultRoleForUser adds all the default non-org-specific role to a user.
func AddDefaultRoleForUser(userID interface{}) {
	enforcer.AddRoleForUser(fmt.Sprint(userID), "default")
}

// AddDefaultRoleForVirtualUser adds org list role to a virtual user.
func AddDefaultRoleForVirtualUser(userID interface{}) {
	enforcer.AddRoleForUser(fmt.Sprint(userID), "defaultVirtual")
}

// AddOrgRoles creates an organization role, by adding the default (*) org policies for the given organization.
func AddOrgRoles(orgids ...uint) {
	basePath := viper.GetString("pipeline.basepath")
	for _, orgid := range orgids {
		enforcer.AddPolicy(orgRoleName(orgid), fmt.Sprintf("%s/api/v1/orgs/%d", basePath, orgid), "*")
		enforcer.AddPolicy(orgRoleName(orgid), fmt.Sprintf("%s/api/v1/orgs/%d/*", basePath, orgid), "*")
		enforcer.AddPolicy(orgRoleName(orgid), fmt.Sprintf("%s/dashboard/orgs/%d/*", basePath, orgid), "*")
	}
}

// AddOrgRoleForUser adds a user to an organization by adding the associated organization role.
func AddOrgRoleForUser(userID interface{}, orgids ...uint) {
	for _, orgid := range orgids {
		enforcer.AddRoleForUser(fmt.Sprint(userID), orgRoleName(orgid))
	}
}

// DeleteOrgRoleForUser removes a user from an organization by removing the associated organization role.
func DeleteOrgRoleForUser(userID uint, orgid uint) {
	enforcer.DeleteRoleForUser(fmt.Sprint(userID), orgRoleName(orgid))
}

// DeleteRolesForUser removes all roles for a given user.
func DeleteRolesForUser(userID uint) {
	enforcer.DeleteUser(fmt.Sprint(userID))
}

func orgRoleName(orgid uint) string {
	return fmt.Sprint("org-", orgid)
}
