package session

import (
	"html/template"
	"net/http"
)

// ManagerInterface session manager interface
type ManagerInterface interface {
	// Add value to session data, if value is not string, will marshal it into JSON encoding and save it into session data.
	Add(w http.ResponseWriter, req *http.Request, key string, value interface{}) error
	// Get value from session data
	Get(req *http.Request, key string) string
	// Pop value from session data
	Pop(w http.ResponseWriter, req *http.Request, key string) string

	// Flash add flash message to session data
	Flash(w http.ResponseWriter, req *http.Request, message Message) error
	// Flashes returns a slice of flash messages from session data
	Flashes(w http.ResponseWriter, req *http.Request) []Message

	// Load get value from session data and unmarshal it into result
	Load(req *http.Request, key string, result interface{}) error
	// PopLoad pop value from session data and unmarshal it into result
	PopLoad(w http.ResponseWriter, req *http.Request, key string, result interface{}) error

	// Middleware returns a new session manager middleware instance.
	Middleware(http.Handler) http.Handler
}

// Message message struct
type Message struct {
	Message template.HTML
	Type    string
}
