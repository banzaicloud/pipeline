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
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	qorauth "github.com/qor/auth"
	"github.com/stretchr/testify/assert"
)

type enforcerStub struct {
	rules []struct {
		userID string
		path   string
		method string
		result bool
	}
}

func (e *enforcerStub) Enforce(org *auth.Organization, user *auth.User, path, method string) (bool, error) {
	for _, rule := range e.rules {
		userID := user.IDString()
		if user.ID == 0 {
			userID = user.Login
		}
		if rule.userID == userID && rule.path == path && (rule.method == method || rule.method == "*") {
			return true, nil
		}
	}

	return false, nil
}

func TestAuthorizationMiddleware_WithBasePath(t *testing.T) {
	e := &enforcerStub{
		rules: []struct {
			userID string
			path   string
			method string
			result bool
		}{
			{
				userID: "1",
				path:   "/path",
				method: "*",
				result: true,
			},
			{
				userID: "virtualUser",
				path:   "/path",
				method: "*",
				result: true,
			},
		},
	}

	tests := map[string]struct {
		basePath     string
		expectedCode int
		user         *auth.User
		method       string
		path         string
	}{
		"with base path": {
			basePath:     "/basePath",
			expectedCode: http.StatusOK,
			user:         &auth.User{ID: 1},
			method:       http.MethodGet,
			path:         "/basePath/path",
		},
		"exact base path": {
			basePath:     "/basePath",
			expectedCode: http.StatusForbidden,
			user:         &auth.User{ID: 1},
			method:       http.MethodGet,
			path:         "/basePathSomething/path",
		},
		"without base path": {
			basePath:     "",
			expectedCode: http.StatusOK,
			user:         &auth.User{ID: 1},
			method:       http.MethodGet,
			path:         "/path",
		},
		"virtual user": {
			basePath:     "",
			expectedCode: http.StatusOK,
			user:         &auth.User{Login: "virtualUser"},
			method:       http.MethodGet,
			path:         "/path",
		},
		"invalid user": {
			basePath:     "",
			expectedCode: http.StatusForbidden,
			user:         &auth.User{ID: 2},
			method:       http.MethodGet,
			path:         "/path",
		},
		"empty user": {
			basePath:     "",
			expectedCode: http.StatusForbidden,
			user:         nil,
			method:       http.MethodGet,
			path:         "/path",
		},
	}

	for name, test := range tests {
		name, test := name, test

		t.Run(name, func(t *testing.T) {
			middleware := NewMiddleware(e, test.basePath, emperror.NewNoopHandler())

			gin.SetMode(gin.ReleaseMode)
			router := gin.New()
			router.Use(middleware)
			router.Handle(test.method, test.path, func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(test.method, test.path, nil)

			req = req.WithContext(context.WithValue(context.Background(), qorauth.CurrentUser, test.user))
			router.ServeHTTP(w, req)

			assert.Equal(t, test.expectedCode, w.Code)
		})
	}
}
