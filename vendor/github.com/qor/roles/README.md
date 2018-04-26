# Roles

Roles is an [authorization](https://en.wikipedia.org/wiki/Authorization) library for [Golang](http://golang.org/), it also integrates nicely with [QOR Admin](http://github.com/qor/admin).

[![GoDoc](https://godoc.org/github.com/qor/roles?status.svg)](https://godoc.org/github.com/qor/roles)

## Usage

### Permission Modes

Permission modes are really the *roles* in [Roles](https://github.com/qor/roles). [Roles](https://github.com/qor/roles) has [5 default permission modes](https://github.com/qor/roles/blob/master/permission.go#L8-L12):

- roles.Read
- roles.Update
- roles.Create
- roles.Delete
- roles.CRUD   // CRUD means Read, Update, Create, Delete

You can use those permission modes, or create your own by [defining permissions](#define-permission).

### Permission Behaviors and Interactions

1. All roles in the Deny mapping for a permission mode are immediately denied without reference to the Allow mapping for that permission mode.

    *E.g.*
    ```go
    roles.Deny(roles.Delete, roles.Anyone).Allow(roles.Delete, "admin")
    ```
     will deny access to `admin` for the permission mode `roles.Delete`, despite the chained call to `Allow()`. *I.e.* `Allow()` has **NO** effect in this chain.

2. If there are **NO** roles in the Allow mapping for a permission mode, then `roles.Anyone` is allowed.

    *E.g.*
    ```go
    roles.Deny(roles.CRUD, "customer")
    ```
    will allow access for permission mode `roles.CRUD` to **any** role that is not a `customer` because the Allow mapping is empty and the blanket allow rule is in force.

3. If even one (1) Allow mapping exists, then only roles on that list will be allowed through.

    *E.g.*
    ```go
    roles.Allow(roles.READ, "admin")
    ```
    allows the `admin` role through and rejects **ALL** other roles.

The following is a flow diagram for a specific permission mode, *e.g.* `roles.READ`.

``` flow
st=>start: Input role
denied0=>end: Denied
allowed0=>end: Allowed
denied1=>end: Denied
allowed1=>end: Allowed
op0=>operation: Exists in Deny map?
op1=>operation: Allow map empty?
op2=>operation: Exists in Allow map?
cond0=>condition: Yes or No?
cond1=>condition: Yes or No?
cond2=>condition: Yes or No?

st->op0->cond0
cond0(no)->op1->cond1
cond0(yes)->denied0
cond1(yes)->allowed0
cond1(no)->op2->cond2
cond2(yes)->allowed1
cond2(no)->denied1
```

Please note that, when using [Roles](https://github.com/qor/roles) with [L10n](http://github.com/qor/l10n). The

```go
// allows the admin role through and rejects ALL other roles.
roles.Allow(roles.READ, "admin")
```

might be invalid because [L10n](http://github.com/qor/l10n) defined a [permission system](http://github.com/qor/l10n#editable-locales) that applys new roles to the current user. For example, There is a user with role "manager", the `EditableLocales` in the [L10n](http://github.com/qor/l10n) permission system returns true in current locale. Then this user actually has two roles "manager" and "locale_admin". because [L10n](http://github.com/qor/l10n) set `resource.Permission.Allow(roles.CRUD, "locale_admin")` to the resource. So the user could access this resource by the role "locale\_admin".

So you either use `Deny` instead which means swtich "white list" to "black list" or make the `EditableLocales` always return blank array which means disabled [L10n](http://github.com/qor/l10n) permission system.

### Define Permission

```go
import "github.com/qor/roles"

func main() {
  // Allow Permission
  permission := roles.Allow(roles.Read, "admin") // `admin` has `Read` permission, `admin` is a role name

  // Deny Permission
  permission := roles.Deny(roles.Create, "user") // `user` has no `Create` permission

  // Using Chain
  permission := roles.Allow(roles.CRUD, "admin").Allow(roles.Read, "visitor") // `admin` has `CRUD` permissions, `visitor` only has `Read` permission
  permission := roles.Allow(roles.CRUD, "admin").Deny(roles.Update, "user") // `admin` has `CRUD` permissions, `user` doesn't has `Update` permission

  // roles `Anyone` means for anyone
  permission := roles.Deny(roles.Update, roles.Anyone) // no one has update permission
}
```

### Check Permission

```go
import "github.com/qor/roles"

func main() {
  permission := roles.Allow(roles.CRUD, "admin").Deny(roles.Create, "manager").Allow(roles.Read, "visitor")

  // check if role `admin` has the Read permission
  permission.HasPermission(roles.Read, "admin")     // => true

  // check if role `admin` has the Create permission
  permission.HasPermission(roles.Create, "admin")     // => true

  // check if role `user` has the Read permission
  permission.HasPermission(roles.Read, "user")     // => true

  // check if role `user` has the Create permission
  permission.HasPermission(roles.Create, "user")     // => false

  // check if role `visitor` has the Read permission
  permission.HasPermission(roles.Read, "user")     // => true

  // check if role `visitor` has the Create permission
  permission.HasPermission(roles.Create, "user")     // => false

  // Check with multiple roles
  // check if role `admin` or `user` has the Create permission
  permission.HasPermission(roles.Create, "admin", "user")     // => true
}
```

### Register Roles

When checking permissions, you will need to know current user's *roles* first. This could quickly get out of hand if you have defined many *roles* based on lots of conditions - so [Roles](https://github.com/qor/roles) provides some helper methods to make it easier:

```go
import "github.com/qor/roles"

func main() {
  // Register roles based on some conditions
  roles.Register("admin", func(req *http.Request, currentUser interface{}) bool {
      return req.RemoteAddr == "127.0.0.1" || (currentUser.(*User) != nil && currentUser.(*User).Role == "admin")
  })

  roles.Register("user", func(req *http.Request, currentUser interface{}) bool {
    return currentUser.(*User) != nil
  })

  roles.Register("visitor", func(req *http.Request, currentUser interface{}) bool {
    return currentUser.(*User) == nil
  })

  // Get roles from a user
  matchedRoles := roles.MatchedRoles(httpRequest, user) // []string{"user", "admin"}

  // Check if role `user` or `admin` has Read permission
  permission.HasPermission(roles.Read, matchedRoles...)
}
```

## License

Released under the [MIT License](http://opensource.org/licenses/MIT).
