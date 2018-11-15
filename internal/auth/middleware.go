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
	"strings"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/gin-gonic/gin"
)

type enforcer interface {
	Enforce(rvals ...interface{}) bool
}

// NewMiddleware returns a new gin middleware that checks user authorization.
func NewMiddleware(e enforcer, basePath string) gin.HandlerFunc {
	m := &middleware{
		enforcer: e,
		basePath: fmt.Sprintf("/%s", strings.Trim(basePath, "/")),
	}

	return func(c *gin.Context) {
		if !m.CheckPermission(c.Request) {
			c.AbortWithStatus(http.StatusForbidden)
		}
	}
}

// middleware stores the casbin handler
type middleware struct {
	enforcer enforcer
	basePath string
}

// getUserID gets the user name from the request.
func (m *middleware) getUserID(r *http.Request) string {
	user := auth.GetCurrentUser(r)
	if user == nil {
		return ""
	}

	if user.ID == 0 {
		return user.Login // This is needed for Drone virtual user tokens
	}

	return user.IDString()
}

// CheckPermission checks the user/method/path combination from the request.
// Returns true (permission granted) or false (permission forbidden)
func (m *middleware) CheckPermission(r *http.Request) bool {
	userID := m.getUserID(r)
	method := r.Method
	path := r.URL.Path

	granted := m.enforcer.Enforce(userID, path, method)

	// Try checking the permission without a base path
	if !granted && m.basePath != "" && strings.HasPrefix(path, fmt.Sprintf("%s/", m.basePath)) {
		granted = m.enforcer.Enforce(userID, strings.TrimPrefix(path, m.basePath), method)
	}

	return granted
}
