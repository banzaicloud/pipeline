[![Build Status](https://travis-ci.org/auth0-community/go-auth0.svg?branch=master)](https://travis-ci.org/auth0-community/go-auth0)
[![Coverage Status](https://coveralls.io/repos/github/auth0-community/go-auth0/badge.svg?branch=master)](https://coveralls.io/github/auth0-community/go-auth0?branch=master)
[![GoDoc](https://godoc.org/github.com/auth0-community/go-auth0?status.png)](https://godoc.org/github.com/auth0-community/go-auth0)
[![Report Cart](http://goreportcard.com/badge/auth0-community/go-auth0)](http://goreportcard.com/report/auth0-community/go-auth0)
[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat)](LICENSE)

# auth0

auth0 is a package helping to authenticate using the [Auth0](https://auth0.com) service.

## Installation 

```
go get github.com/auth0-community/go-auth0
```

## Client Credentials - HS256
Using HS256, the validation key is the secret you retrieve in the dashboard.
```go
// Creates a configuration with the Auth0 information
secret, _ := base64.URLEncoding.DecodeString(os.Getenv("AUTH0_CLIENT_SECRET"))
secretProvider := auth0.NewKeyProvider(secret)
audience := os.Getenv("AUTH0_CLIENT_ID")

configuration := auth0.NewConfiguration(secretProvider, []string{audience}, "https://mydomain.eu.auth0.com/", jose.HS256)
validator := auth0.NewValidator(configuration)

token, err := validator.ValidateRequest(r)

if err != nil {
    fmt.Println("Token is not valid:", token)
}
```

## Client Credentials - RS256
Using RS256, the validation key is the certificate you find in advanced settings

```go
// Extracted from https://github.com/square/go-jose/blob/master/utils.go
// LoadPublicKey loads a public key from PEM/DER-encoded data.
func LoadPublicKey(data []byte) (interface{}, error) {
	input := data

	block, _ := pem.Decode(data)
	if block != nil {
		input = block.Bytes
	}

	// Try to load SubjectPublicKeyInfo
	pub, err0 := x509.ParsePKIXPublicKey(input)
	if err0 == nil {
		return pub, nil
	}

	cert, err1 := x509.ParseCertificate(input)
	if err1 == nil {
		return cert.PublicKey, nil
	}

	return nil, fmt.Errorf("square/go-jose: parse error, got '%s' and '%s'", err0, err1)
}
// Create a configuration with the Auth0 information
secret, _ := LoadPublicKey(sharedKey)
secretProvider := auth0.NewKeyProvider(secret)
audience := os.Getenv("AUTH0_CLIENT_ID")

configuration := auth0.NewConfiguration(secretProvider, []string{audience}, "https://mydomain.eu.auth0.com/", jose.RS256)
validator := auth0.NewValidator(configuration)

token, err := validator.ValidateRequest(r)

if err != nil {
    fmt.Println("Token is not valid:", token)
}
```
## API with JWK

```go
   
client := NewJWKClient(JWKClientOptions{URI: "https://mydomain.eu.auth0.com/.well-known/jwks.json"})
audience := os.Getenv("AUTH0_CLIENT_ID")
configuration := NewConfiguration(client, []string{audience}, "https://mydomain.eu.auth0.com/", jose.RS256)
validator := NewValidator(configuration)

token, err := validator.ValidateRequest(r)

if err != nil {
    fmt.Println("Token is not valid:", token)
}
```
## Example

### Gin

Using [Gin](https://github.com/gin-gonic/gin) and the [Auth0 Authorization Extension](https://auth0.com/docs/extensions/authorization-extension), you 
may want to implement the authentication auth like the following:

```go
var auth.AdminGroup string = "my_admin_group"

// Access Control Helper function.
func shouldAccess(wantedGroups []string, groups []interface{}) bool { 
 /* Fill depending on your needs */
}

// Wrapping a Gin endpoint with Auth0 Groups.
func Auth0Groups(wantedGroups ...string) gin.HandlerFunc {

	return gin.HandlerFunc(func(c *gin.Context) {

		tok, err := validator.ValidateRequest(c.Request)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			log.Println("Invalid token:", err)
			return
		}

		claims := map[string]interface{}{}
		err = validator.Claims(c.Request, tok, &claims)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid claims"})
			c.Abort()
			log.Println("Invalid claims:", err)
			return
		}

		metadata, okMetadata := claims["app_metadata"].(map[string]interface{})
		authorization, okAuthorization := metadata["authorization"].(map[string]interface{})
		groups, hasGroups := authorization["groups"].([]interface{})
		if !okMetadata || !okAuthorization || !hasGroups || !shouldAccess(wantedGroups, groups) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "need more privileges"})
			c.Abort()
			log.Println("Need more provileges")
			return
		}
		c.Next()
	})
}

// Use it
r.PUT("/news", auth.Auth0Groups(auth.AdminGroup), api.GetNews)
```

For a sample usage, take a look inside the `example` directory.
