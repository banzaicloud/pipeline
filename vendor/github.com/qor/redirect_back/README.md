# Redirect Back

A Golang HTTP Handler that redirect back to last URL saved in session

## Usage

```go
package main

import (
	"net/http"

	"github.com/qor/redirect_back"
	"github.com/qor/session/manager"
)

var RedirectBack = redirect_back.New(&redirect_back.Config{
	IgnoredPaths:      []string{"/login"},     // Will ignore requests that has those paths when set return path
	IgnoredPrefixes:   []string{"/auth"},      // Will ignore requests that has those prefixes when set return path
	AllowedExtensions: []string{"", ".html"}   // Only save requests w/o extension or extension `.html` (default setting)
	IgnoreFunc: func(req *http.Request) bool { // Will ignore request if `IgnoreFunc` returns true
		return false
	},
	SessionManager: manager.SessionManager,
	FallbackPath:   "/",
})

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/page", pageHandler)

	// Wrap your application's handlers or router with redirect back's middleware
	http.ListenAndServe(":8000", manager.SessionManager.Middleware(RedirectBack.Middleware(mux)))
}

func homeHandler(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("home"))
}

func pageHandler(w http.ResponseWriter, req *http.Request) {
	// Redirect to return path or the default one
	RedirectBack.RedirectBack(w, req)
}
```
