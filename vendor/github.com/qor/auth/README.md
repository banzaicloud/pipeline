# Auth

Auth is a modular authentication system for web development in Golang, it provides different authentication backends to accelerate your development.

Currently Auth has database password, github, google, facebook, twitter authentication support, and it is fairly easy to add other support based on [Auth's Provider interface](https://godoc.org/github.com/qor/auth#Provider)

## Quick Start

Auth aims to provide an easy to use authentication system that don't require much developer's effort.

To use it, basic flow is:

* Initialize Auth with configuration
* Register some providers
* Register it into router

Here is an example:

```go
import (
  "github.com/qor/auth"
  "github.com/qor/auth/auth_identity"
  "github.com/qor/auth/providers/github"
  "github.com/qor/auth/providers/google"
  "github.com/qor/auth/providers/password"
  "github.com/qor/auth/providers/facebook"
  "github.com/qor/auth/providers/twitter"
  "github.com/qor/session/manager"
)

var (
  // Initialize gorm DB
  gormDB, _ = gorm.Open("sqlite3", "sample.db")

  // Initialize Auth with configuration
  Auth = auth.New(&auth.Config{
    DB: gormDB,
  })
)

func init() {
  // Migrate AuthIdentity model, AuthIdentity will be used to save auth info, like username/password, oauth token, you could change that.
  gormDB.AutoMigrate(&auth_identity.AuthIdentity{})

  // Register Auth providers
  // Allow use username/password
  Auth.RegisterProvider(password.New(&password.Config{}))

  // Allow use Github
  Auth.RegisterProvider(github.New(&github.Config{
    ClientID:     "github client id",
    ClientSecret: "github client secret",
  }))

  // Allow use Google
  Auth.RegisterProvider(google.New(&google.Config{
    ClientID:     "google client id",
    ClientSecret: "google client secret",
  }))

  // Allow use Facebook
  Auth.RegisterProvider(facebook.New(&facebook.Config{
    ClientID:     "facebook client id",
    ClientSecret: "facebook client secret",
  }))

  // Allow use Twitter
  Auth.RegisterProvider(twitter.New(&twitter.Config{
    ClientID:     "twitter client id",
    ClientSecret: "twitter client secret",
  }))
}

func main() {
  mux := http.NewServeMux()

  // Mount Auth to Router
  mux.Handle("/auth/", Auth.NewServeMux())
  http.ListenAndServe(":9000", manager.SessionManager.Middleware(mux))
}
```

That's it, then you could goto `http://127.0.0.1:9000/auth/login` to try Auth features, like login, logout, register, forgot/change password...

And it could be even easier with [Auth Themes](#auth-themes), you could integrate Auth into your application with few line configurations.

## Usage

Auth has many configurations that could be used to customize it for different usage, lets start from Auth's [Config](http://godoc.org/github.com/qor/auth#Config).

### Models

Auth has two models, model `AuthIdentityModel` is used to save login information, model `UserModel` is used to save user information.

The reason we save auth and user info into two different models, as we want to be able to link a user to mutliple auth info records, so a user could have multiple ways to login.

If this is not required for you, you could just set those two models to same one or skip set `UserModel`.

* `AuthIdentityModel`

Different provider usually use different information to login, like provider `password` use username/password, `github` use github user ID, so for each provider, it will save those information into its own record.

You are not necessary to set `AuthIdentityModel`, Auth has a default [definition of AuthIdentityModel](http://godoc.org/github.com/qor/auth/auth_identity#AuthIdentity), in case of you want to change it, make sure you have [auth_identity.Basic](http://godoc.org/github.com/qor/auth/auth_identity#Basic) embedded, as `Auth` assume you have same data structure in your database, so it could query/create records with SQL.

* `UserModel`

By default, there is no `UserModel` defined, even though, you still be able to use `Auth` features, `Auth` will return used auth info record as logged user.

But usually your application will have a `User` model, after you set its value, when you register a new account from any provider, Auth will create/get a user with `UserStorer`, and link its ID to the auth identity record.

### Customize views

Auth using [Render](http://github.com/qor/render) to render pages, you could refer it for how to register func maps, extend views paths, also be sure to refer [BindataFS](https://github.com/qor/bindatafs) if you want to compile your application into a binary.

If you want to preprend view paths, you could add them to `ViewPaths`, which would be helpful if you want to overwrite the default (ugly) login/register pages or develop auth themes like [https://github.com/qor/auth_themes](https://github.com/qor/auth_themes)

### Sending Emails

Auth using [Mailer](http://github.com/qor/mailer) to send emails, by default, Auth will print emails to console, please configure it to send real one.

### User Storer

Auth created a default UserStorer to get/save user based on your `AuthIdentityModel`, `UserModel`'s definition, in case of you want to change it, you could implement your own [User Storer](http://godoc.org/github.com/qor/auth#UserStorerInterface)

### Session Storer

Auth also has a default way to handle sessions, flash messages, which could be overwrited by implementing [Session Storer Interface](http://godoc.org/github.com/qor/auth#SessionStorerInterface).

By default, Auth is using [session](https://github.com/qor/session)'s default manager to save data into cookies, but in order to save cookies correctly, you have to register session's Middleware into your router, e.g:

```go
func main() {
	mux := http.NewServeMux()

	// Register Router
	mux.Handle("/auth/", Auth.NewServeMux())
	http.ListenAndServe(":9000", manager.SessionManager.Middleware(mux))
}
```

### Redirector

After some Auth actions, like logged, registered or confirmed, Auth will redirect user to some URL, you could configure which page to redirect with `Redirector`, by default, will redirct to home page.

If you want to redirect to last visited page, [redirect_back](https://github.com/qor/redirect_back) is for you, you could configure it and use it as the Redirector, like:

```go
var RedirectBack = redirect_back.New(&redirect_back.Config{
	SessionManager:  manager.SessionManager,
	IgnoredPrefixes: []string{"/auth"},
}

var Auth = auth.New(&auth.Config{
	...
	Redirector: auth.Redirector{RedirectBack},
})
```

BTW, to make it works correctly, `redirect_back` need to save last visisted URL into session with session manager for each request, that's means, you need to mount `redirect_back`, and `SessionManager`'s middleware into router.

```go
http.ListenAndServe(":9000", manager.SessionManager.Middleware(RedirectBack.Middleware(mux)))
```

## Advanced Usage

### Auth Themes

In order to save more developer's effort, we have created some [auth themes](https://github.com/qor/auth_themes).

It usually has well designed pages, if you don't much custom requirements, you could just have few lines to make Auth system ready to use for your application, for example:

```go
import "github.com/qor/auth_themes/clean"

var Auth = clean.New(&auth.Config{
	DB:         db.DB,
	Render:     config.View,
	Mailer:     config.Mailer,
	UserModel:  models.User{},
})
```

Check Auth Theme's [document](https://github.com/qor/auth_themes) for How To use/create Auth themes

### Authorization

`Authentication` is the process of verifying who you are, `Authorization` is the process of verifying that you have access to something.

Auth package not only provides `Authentication`, but also `Authorization`, please checkout [authority](https://github.com/qor/auth/tree/master/authority) for more details
