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
	"github.com/goph/emperror"
)

// Enforcer checks if the current user has access to the organization resource under path with method
type Enforcer interface {
	Enforce(org *auth.Organization, user *auth.User, path, method string) (bool, error)
}

// NewMiddleware returns a new gin middleware that checks user authorization.
func NewMiddleware(e Enforcer, basePath string, errorHandler emperror.Handler) gin.HandlerFunc {
	m := &middleware{
		enforcer:     e,
		basePath:     fmt.Sprintf("/%s", strings.Trim(basePath, "/")),
		errorHandler: errorHandler,
	}

	return func(c *gin.Context) {
		granted, err := m.CheckPermission(c.Request)
		if err != nil {
			err = emperror.Wrap(err, "failed to check permissions for request")
			errorHandler.Handle(err)
			c.AbortWithError(http.StatusInternalServerError, err)
		} else if !granted {
			c.AbortWithStatus(http.StatusForbidden)
		}
	}
}

// middleware wraps an Enforcer to make it gin-ish
type middleware struct {
	enforcer     Enforcer
	basePath     string
	errorHandler emperror.Handler
}

// CheckPermission checks the user/method/path combination from the request.
// Returns true (permission granted) or false (permission forbidden)
func (m *middleware) CheckPermission(r *http.Request) (bool, error) {
	org := auth.GetCurrentOrganization(r)
	user := auth.GetCurrentUser(r)
	method := r.Method
	path := r.URL.Path

	if user == nil {
		return false, nil
	}

	granted, err := m.enforcer.Enforce(org, user, path, method)
	if err != nil {
		return granted, err
	}

	// Try checking the permission without a base path
	if !granted && m.basePath != "" && strings.HasPrefix(path, fmt.Sprintf("%s/", m.basePath)) {
		granted, err = m.enforcer.Enforce(org, user, strings.TrimPrefix(path, m.basePath), method)
	}

	return granted, err
}
