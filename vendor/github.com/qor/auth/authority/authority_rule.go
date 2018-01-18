package authority

import (
	"net/http"
	"time"

	"github.com/qor/roles"
)

// Rule authority rule's definition
type Rule struct {
	TimeoutSinceLastLogin            time.Duration
	LongestDistractionSinceLastLogin time.Duration
}

// Handler generate roles checker
func (authority Authority) Handler(rule Rule) roles.Checker {
	return func(req *http.Request, user interface{}) bool {
		claims, _ := authority.Auth.Get(req)

		// Check Last Auth
		if rule.TimeoutSinceLastLogin > 0 {
			if claims == nil || claims.LastLoginAt == nil || time.Now().Add(-rule.TimeoutSinceLastLogin).After(*claims.LastLoginAt) {
				return false
			}
		}

		// Check Distraction
		if rule.LongestDistractionSinceLastLogin > 0 {
			if claims == nil || claims.LongestDistractionSinceLastLogin == nil || *claims.LongestDistractionSinceLastLogin > rule.LongestDistractionSinceLastLogin {
				return false
			}
		}

		return true
	}
}

// Register register authority rule into Role
func (authority *Authority) Register(name string, rule Rule) {
	authority.Config.Role.Register(name, authority.Handler(rule))
}

// Allow Check allow role or not
func (authority *Authority) Allow(role string, req *http.Request) bool {
	currentUser := authority.Auth.GetCurrentUser(req)
	return authority.Role.HasRole(req, currentUser, role)
}
