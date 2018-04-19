package auth

import (
	"fmt"
	"net/http"
	"time"

	"github.com/banzaicloud/pipeline/model"
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
func NewAuthorizer() gin.HandlerFunc {
	dbName := viper.GetString("database.dbname")
	adapter := gormadapter.NewAdapter("mysql", model.GetDataSource(dbName), true)
	model := casbin.NewModel(modelDefinition)
	enforcer = casbin.NewSyncedEnforcer(model, adapter, logging)
	enforcer.StartAutoLoadPolicy(10 * time.Second)
	AddDefaultPolicies()
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

// GetUserName gets the user name from the request.
// Currently, only HTTP Bearer token authentication is supported
func (a *BearerAuthorizer) GetUserName(r *http.Request) string {
	user := GetCurrentUser(r)
	return user.Login
}

// CheckPermission checks the user/method/path combination from the request.
// Returns true (permission granted) or false (permission forbidden)
func (a *BearerAuthorizer) CheckPermission(r *http.Request) bool {
	user := a.GetUserName(r)
	method := r.Method
	path := r.URL.Path
	return a.enforcer.Enforce(user, path, method)
}

// RequirePermission returns the 403 Forbidden to the client
func (a *BearerAuthorizer) RequirePermission(c *gin.Context) {
	c.AbortWithStatus(http.StatusForbidden)
}

func AddDefaultPolicies() {
	basePath := viper.GetString("pipeline.basepath")
	enforcer.AddPolicy("default", basePath+"/api/v1/orgs", "*")
	enforcer.AddPolicy("default", basePath+"/api/v1/token", "*") // DEPRECATED
	enforcer.AddPolicy("default", basePath+"/api/v1/tokens", "*")
}

func AddDefaultPolicyToUser(username string) {
	enforcer.AddRoleForUser(username, "default")
}

func AddOrgRoles(orgids ...uint) {
	basePath := viper.GetString("pipeline.basepath")
	for _, orgid := range orgids {
		enforcer.AddPolicy(orgRoleName(orgid), fmt.Sprintf("%s/api/v1/orgs/%d", basePath, orgid), "*")
		enforcer.AddPolicy(orgRoleName(orgid), fmt.Sprintf("%s/api/v1/orgs/%d/*", basePath, orgid), "*")
	}
}

func AddOrgRoleToUser(username string, orgids ...uint) {
	for _, orgid := range orgids {
		enforcer.AddRoleForUser(username, orgRoleName(orgid))
	}
}

func DeleteOrgRoleFromUser(username string, orgid uint) {
	enforcer.DeleteRoleForUser(username, orgRoleName(orgid))
}

func orgRoleName(orgid uint) string {
	return fmt.Sprint("org-", orgid)
}
