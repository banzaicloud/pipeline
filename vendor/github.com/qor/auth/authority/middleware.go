package authority

import (
	"net/http"
	"time"

	"github.com/qor/qor/utils"
)

// ClaimsContextKey authority claims key
var ClaimsContextKey utils.ContextKey = "authority_claims"

// Middleware authority middleware used to record activity time
func (authority *Authority) Middleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if claims, err := authority.Auth.Get(req); err == nil {
			var zero time.Duration

			lastActiveAt := claims.LastActiveAt
			if lastActiveAt != nil {
				lastDistractionTime := time.Now().Sub(*lastActiveAt)
				if claims.LongestDistractionSinceLastLogin == nil || *claims.LongestDistractionSinceLastLogin < lastDistractionTime {
					claims.LongestDistractionSinceLastLogin = &lastDistractionTime
				}

				if claims.LastLoginAt != nil {
					if claims.LastLoginAt.After(*claims.LastActiveAt) {
						claims.LongestDistractionSinceLastLogin = &zero
					} else if loggedDuration := claims.LastActiveAt.Sub(*claims.LastLoginAt); *claims.LongestDistractionSinceLastLogin > loggedDuration {
						claims.LongestDistractionSinceLastLogin = &loggedDuration
					}
				}
			} else {
				claims.LongestDistractionSinceLastLogin = &zero
			}

			now := time.Now()
			claims.LastActiveAt = &now

			authority.Auth.Update(w, req, claims)
		}

		handler.ServeHTTP(w, req)
	})
}
