package auth

import (
	"net/http"
	"strings"

	"github.com/jinzhu/gorm"
)

// IsHttps is a helper function that evaluates the http.Request
// and returns True if the Request uses HTTPS. It is able to detect,
// using the X-Forwarded-Proto, if the original request was HTTPS and
// routed through a reverse proxy with SSL termination.
func IsHttps(r *http.Request) bool {
	switch {
	case r.URL.Scheme == "https":
		return true
	case r.TLS != nil:
		return true
	case strings.HasPrefix(r.Proto, "HTTPS"):
		return true
	case r.Header.Get("X-Forwarded-Proto") == "https":
		return true
	default:
		return false
	}
}

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
		Domain:   cookieDomain,
		HttpOnly: SessionCookieHTTPOnly,
		Secure:   IsHttps(r),
		MaxAge:   SessionCookieMaxAge,
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
