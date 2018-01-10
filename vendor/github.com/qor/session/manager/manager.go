package manager

import (
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/qor/middlewares"
	"github.com/qor/session"
	"github.com/qor/session/gorilla"
)

// SessionManager default session manager
var SessionManager session.ManagerInterface = gorilla.New("_session", sessions.NewCookieStore([]byte("secret")))

func init() {
	middlewares.Use(middlewares.Middleware{
		Name: "session",
		Handler: func(handler http.Handler) http.Handler {
			return SessionManager.Middleware(handler)
		},
	})
}
