# Middleware Stack

Manage Golang HTTP middlewares

## Usage

```go
func main() {
	Stack := &MiddlewareStack{}

	// Add middleware `auth` to stack
	Stack.Use(&middlewares.Middleware{
		Name: "auth",
		// Insert middleware `auth` after middleware `session` if it exists
		InsertAfter: []string{"session"},
		// Insert middleware `auth` before middleare `authorization` if it exists
		InsertBefore: []string{"authorization"},
	})

	// Remove middleware `cookie` from stack
	Stack.Remove("cookie")

	mux := http.NewServeMux()
	http.ListenAndServe(":9000", Stack.Apply(mux))
}
```

## Default Middleware Stack

`DefaultMiddlewareStack` is an initialized middleware stack, It is defined like this:

```go
var DefaultMiddlewareStack = &MiddlewareStack{}
```

There are some global methods could be used to manage its middlewares, e.g:

```go
func main() {
	// Add middleware `auth` to default stack
	middlewares.Use(&middlewares.Middleware{
		Name: "auth",
		// Insert middleware `auth` after middleware `session` if it exists
		InsertAfter: []string{"session"},
		// Insert middleware `auth` before middleare `authorization` if it exists
		InsertBefore: []string{"authorization"},
	})

	// Remove middleware `cookie` from default stack
	middlewares.Remove("cookie")

	mux := http.NewServeMux()
	http.ListenAndServe(":9000", middlewares.Apply(mux))
}
```

### QOR Integration

There are many QOR libraries that requires to regsiter middlewares, it will register its middleware to the `DefaultMiddlewareStack`, so if you don't want to manage those libraries's middlewares by yourself, you could just used the `DefaultMiddlewareStack` for your application.
