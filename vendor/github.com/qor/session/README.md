# Session

Session Management for QOR

It wrapped other libs like [SCS](https://github.com/alexedwards/scs), [Gorilla Session](http://www.gorillatoolkit.org/pkg/sessions) into a common interface, which will be used for QOR libs and your application.

## Basic Usage

```go
import (
	"github.com/gorilla/sessions"
	"github.com/qor/session/gorilla"
	// "github.com/alexedwards/scs/engine/memstore"
)

var SessionManager = session.ManagerInterface

func main() {
	// Use gorilla session as the backend
	engine := sessions.NewCookieStore([]byte("something-very-secret"))
	SessionManager = gorilla.New("_session", engine)
	// Use SCS as the backend
	// engine := memstore.New(0)
	// SessionManager := scs.New(engine)

	mux := http.NewServeMux()
	mux.HandleFunc("/put", putHandler)
	mux.HandleFunc("/get", getHandler)
	// Your routes

	// Wrap your application's handlers or router with session manager's middleware
	http.ListenAndServe(":7000", manager.Middleware(mux))
}

func putHandler(w http.ResponseWriter, req *http.Request) {
	// Store a key and associated value into session data
	SessionManager.Add(w, req, "key", "value")
}

func getHandler(w http.ResponseWriter, req *http.Request) {
	// Get saved session data with key
	value := SessionManager.Get(req, "key")
	io.WriteString(w, value)
}
```

## Session Manager's Interface

```go
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
```

## QOR Integration

We have created a default session manager in package `github.com/qor/session/manager`, which is used in some QOR libs like QOR Admin, QOR Auth by default to manage session, flash messages.

It is defined like below:

```go
var SessionManager session.ManagerInterface = gorilla.New("_session", sessions.NewCookieStore([]byte("secret")))
```

You should change it to your own session storage or use your own secret code.

```go
import (
	"github.com/qor/session/manager"
)

func main() {
	// Overwrite session manager
	engine := sessions.NewCookieStore([]byte("your-own-secret-code"))
	manager.SessionManager = gorilla.New("_gorilla_session", engine)
}
```

## License

Released under the [MIT License](http://opensource.org/licenses/MIT).
