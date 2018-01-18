package roles

import "net/http"

// Global global role instance
var Global = &Role{}

// Register register role with conditions
func Register(name string, fc Checker) {
	Global.Register(name, fc)
}

// Allow allows permission mode for roles
func Allow(mode PermissionMode, roles ...string) *Permission {
	return Global.Allow(mode, roles...)
}

// Deny deny permission mode for roles
func Deny(mode PermissionMode, roles ...string) *Permission {
	return Global.Deny(mode, roles...)
}

// Get role defination
func Get(name string) (Checker, bool) {
	return Global.Get(name)
}

// Remove role definition from global role instance
func Remove(name string) {
	Global.Remove(name)
}

// Reset role definitions from global role instance
func Reset() {
	Global.Reset()
}

// MatchedRoles return defined roles from user
func MatchedRoles(req *http.Request, user interface{}) []string {
	return Global.MatchedRoles(req, user)
}

// HasRole check if current user has role
func HasRole(req *http.Request, user interface{}, roles ...string) bool {
	return Global.HasRole(req, user)
}

// NewPermission initialize a new permission for default role
func NewPermission() *Permission {
	return Global.NewPermission()
}
