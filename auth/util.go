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
	"net/http"

	"github.com/gin-gonic/gin/render"
	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/pkg/common"
)

// SetCookie writes the cookie value.
func SetCookie(w http.ResponseWriter, r *http.Request, name, value string) {
	cookieDomain := r.URL.Host
	if CookieDomain != "" {
		cookieDomain = CookieDomain
	}

	cookie := http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: SessionCookieHTTPOnly,
		Secure:   global.Config.Auth.Cookie.Secure,
		MaxAge:   SessionCookieMaxAge,
	}

	if global.Config.Auth.Cookie.SetDomain {
		cookie.Domain = cookieDomain
	}

	http.SetCookie(w, &cookie)
}

// DelCookie deletes a cookie.
func DelCookie(w http.ResponseWriter, r *http.Request, name string) {
	cookieDomain := r.URL.Host
	if CookieDomain != "" {
		cookieDomain = CookieDomain
	}

	cookie := http.Cookie{
		Name:   name,
		Value:  "deleted",
		Path:   "/",
		Domain: cookieDomain,
		MaxAge: -1,
	}

	http.SetCookie(w, &cookie)
}

// GormErrorToStatusCode translates GORM errors to HTTP status codes
func GormErrorToStatusCode(err error) int {
	if err == gorm.ErrRecordNotFound {
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}

func httpJSONError(w http.ResponseWriter, err error, code int) {
	render := render.JSON{Data: common.ErrorResponse{Error: err.Error()}}
	render.WriteContentType(w)
	w.WriteHeader(code)
	_ = render.Render(w)
}
