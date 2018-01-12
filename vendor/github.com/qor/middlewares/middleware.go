package middlewares

import "net/http"

// Middleware middleware struct
type Middleware struct {
	Name         string
	Handler      func(http.Handler) http.Handler
	InsertAfter  []string
	InsertBefore []string
	Requires     []string
}
