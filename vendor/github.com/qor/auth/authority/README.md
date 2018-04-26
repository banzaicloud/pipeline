# Authority

Authority is an authorization package for Golang.

When building web applications, we usually have requirements like verify current user has access to something or not, Authority could help you by providing HTTP middleware and helper method.

It is based on [Roles](http://github.com/qor/roles), means to define an ability, you need to register a role with `Roles`, refer [Roles](http://github.com/qor/roles) for how to do that.

## Usage

### Initialize Authority

Authority could use with [Auth](http://github.com/qor/auth) to get current user, handle sessions, you could use it, or implement your own [AuthInterface](http://godoc.org/github.com/qor/auth/authority#AuthInterface)

```go
import (
  "github.com/qor/auth"
  "github.com/qor/auth/authority"
  "github.com/qor/roles"
)

func main() {
  Auth := auth.New(&auth.Config{})

  Authority := authority.New(&authority.Config{
    Auth: Auth,
    Role: roles.Global, // default configuration
    AccessDeniedHandler: func(w http.ResponseWriter, req *http.Request) { // redirect to home page by default
      http.Redirect(w, req, "/", http.StatusSeeOther)
    },
  })
}
```

## Defining Abilities

Refer [Roles](http://github.com/qor/roles) for how to use roles to register roles, here is a sample:

```go
roles.Register("admin", func(req *http.Request, currentUser interface{}) bool {
  return req.RemoteAddr == "127.0.0.1" || (currentUser.(*User) != nil && currentUser.(*User).Role == "admin")
})
```

You might have some requirements like time based authorization, for example:

* I get distracted, come back to the site in 2 hours, then no access to my account details page, but still be able to visit shopping cart
* When place an order, I have to been authorized less than 60 minutes

Authority provides some [Rules](http://godoc.org/github.com/qor/auth/authority#Rule) to make you define them easily, used like:

```go
Authority.Register("access_account_pages", authority.Rule{
  TimeoutSinceLastActive: 2 * time.Hour,
})

Authority.Register("place_an_order", authority.Rule{
  TimeoutSinceLastAuth: time.Hour,
})
```

## Authorization Middleware

```go
func main() {
  mux := http.NewServeMux()

  // Require current user has `access_account_pages` ability to acccess `AccountProfileHandler`
  mux.Handle("/account/profile", Authority.Authorize("access_account_pages")(AccountProfileHandler))

  // Any logged user could acccess `AccountProfileHandler` if no roles specfied
  mux.Handle("/account/profile", Authority.Authorize()(AccountProfileHandler))

  http.ListenAndServe(":9000", mux)
}

func AccountProfileHandler(w http.ResponseWriter, req *http.Request) {
  // ...
}
```

## Authorization Helper Method

```go
func updateCreditCard(w http.ResponseWriter, req *http.Request) {
  if Authority.Allow("place_an_order", req) {
    // do something
  } else {
    // do something
  }
}
```
