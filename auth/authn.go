package auth

import (
	"encoding/base32"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	bauth "github.com/banzaicloud/bank-vaults/auth"
	"github.com/banzaicloud/pipeline/config"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/qor/auth"
	"github.com/qor/auth/auth_identity"
	"github.com/qor/auth/claims"
	"github.com/qor/auth/providers/github"
	"github.com/qor/redirect_back"
	"github.com/qor/session"
	"github.com/qor/session/gorilla"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// PipelineSessionCookie holds the name of the Cookie Pipeline sets in the browser
const PipelineSessionCookie = "_banzai_session"

// DroneSessionCookie holds the name of the Cookie Drone sets in the browser
const DroneSessionCookie = "user_sess"

// DroneSessionCookieType is the Drone token type used for browser sessions
const DroneSessionCookieType = "sess"

// DroneUserTokenType is the Drone token type used for API sessions
const DroneUserTokenType bauth.TokenType = "user"

// DroneHookTokenType is the Drone token type used for API sessions
const DroneHookTokenType bauth.TokenType = "hook"

// For all Drone token types please see: https://github.com/drone/drone/blob/master/shared/token/token.go#L12

// SessionCookieMaxAge holds long an authenticated session should be valid in seconds
const SessionCookieMaxAge = 30 * 24 * 60 * 60

// SessionCookieHTTPOnly describes if the cookies should be accessible from HTTP requests only (no JS)
const SessionCookieHTTPOnly = true

// Init authorization
var (
	log *logrus.Logger

	DroneDB *gorm.DB

	redirectBack *redirect_back.RedirectBack
	Auth         *auth.Auth

	signingKey       string
	signingKeyBase32 string
	TokenStore       bauth.TokenStore

	// JwtIssuer ("iss") claim identifies principal that issued the JWT
	JwtIssuer string

	// JwtAudience ("aud") claim identifies the recipients that the JWT is intended for
	JwtAudience string

	// CookieDomain is the domain field for cookies
	CookieDomain string

	// Handler is the Gin authentication middleware
	Handler gin.HandlerFunc

	// SessionManager is responsible for handling browser session Cookies
	SessionManager session.ManagerInterface
)

// Simple init for logging
func init() {
	log = config.Logger()
}

// DroneClaims struct to store the drone claim related things
type DroneClaims struct {
	*claims.Claims
	Type bauth.TokenType `json:"type,omitempty"`
	Text string          `json:"text,omitempty"`
}

// Init initializes the auth
func Init(db *gorm.DB) {
	JwtIssuer = viper.GetString("auth.jwtissuer")
	JwtAudience = viper.GetString("auth.jwtaudience")
	CookieDomain = viper.GetString("auth.cookieDomain")

	signingKey = viper.GetString("auth.tokensigningkey")
	if signingKey == "" {
		panic("Token signing key is missing from configuration")
	}
	if len(signingKey) < 32 {
		panic("Token signing key must be at least 32 characters")
	}

	signingKeyBytes := []byte(signingKey)
	signingKeyBase32 = base32.StdEncoding.EncodeToString(signingKeyBytes)

	cookieAuthenticationKey := signingKeyBytes
	cookieEncryptionKey := signingKeyBytes[:32]

	cookieStore := sessions.NewCookieStore(cookieAuthenticationKey, cookieEncryptionKey)
	cookieStore.Options.MaxAge = SessionCookieMaxAge
	cookieStore.Options.HttpOnly = SessionCookieHTTPOnly
	cookieStore.Options.Secure = viper.GetBool("auth.secureCookie")
	if CookieDomain != "" {
		cookieStore.Options.Domain = CookieDomain
	}

	SessionManager = gorilla.New(PipelineSessionCookie, cookieStore)

	// A RedirectBack instance which constantly redirects to /ui
	redirectBack = redirect_back.New(&redirect_back.Config{
		SessionManager:  SessionManager,
		IgnoredPrefixes: []string{"/"},
		IgnoreFunc: func(r *http.Request) bool {
			return true
		},
		FallbackPath: viper.GetString("pipeline.uipath"),
	})

	DroneDB = db

	// Initialize Auth with configuration
	Auth = auth.New(&auth.Config{
		DB:                config.DB(),
		Redirector:        auth.Redirector{RedirectBack: redirectBack},
		AuthIdentityModel: AuthIdentity{},
		UserModel:         User{},
		ViewPaths:         []string{"views"},
		SessionStorer: &BanzaiSessionStorer{
			SessionStorer: auth.SessionStorer{
				SessionName:    "_auth_session",
				SessionManager: SessionManager,
				SigningMethod:  jwt.SigningMethodHS256,
				SignedString:   signingKeyBase32,
			},
		},
		UserStorer:        BanzaiUserStorer{signingKeyBase32: signingKeyBase32, droneDB: DroneDB},
		LogoutHandler:     BanzaiLogoutHandler,
		DeregisterHandler: BanzaiDeregisterHandler,
	})

	githubProvider := github.New(&github.Config{
		// ClientID and ClientSecret is validated inside github.New()
		ClientID:     viper.GetString("auth.clientid"),
		ClientSecret: viper.GetString("auth.clientsecret"),

		// The same as Drone's scopes
		Scopes: []string{
			"repo",
			"user:email",
			"read:org",
		},
	})
	githubProvider.AuthorizeHandler = NewGithubAuthorizeHandler(githubProvider)
	Auth.RegisterProvider(githubProvider)

	TokenStore = bauth.NewVaultTokenStore("pipeline")

	jwtAuth := bauth.JWTAuth(TokenStore, signingKey, func(claims *bauth.ScopedClaims) interface{} {
		userID, _ := strconv.ParseUint(claims.Subject, 10, 32)
		return &User{
			ID:      uint(userID),
			Login:   claims.Text, // This is needed for Drone virtual user tokens
			Virtual: claims.Type == DroneHookTokenType,
		}
	})

	Handler = func(c *gin.Context) {
		currentUser := Auth.GetCurrentUser(c.Request)
		if currentUser != nil {
			return
		}
		jwtAuth(c)
	}
}

// Install the whole OAuth and JWT Token based authn/authz mechanism to the specified Gin Engine.
func Install(engine *gin.Engine) {

	// We have to make the raw net/http handlers a bit Gin-ish
	authHandler := gin.WrapH(Auth.NewServeMux())
	engine.Use(gin.WrapH(SessionManager.Middleware(utils.NopHandler{})))
	engine.Use(gin.WrapH(redirectBack.Middleware(utils.NopHandler{})))

	authGroup := engine.Group("/auth/")
	{
		authGroup.GET("/login", authHandler)
		authGroup.GET("/logout", authHandler)
		authGroup.GET("/register", authHandler)
		authGroup.POST("/deregister", authHandler)
		authGroup.GET("/github/login", authHandler)
		authGroup.GET("/github/logout", authHandler)
		authGroup.GET("/github/register", authHandler)
		authGroup.GET("/github/callback", authHandler)
		authGroup.POST("/tokens", GenerateToken)
		authGroup.GET("/tokens", GetTokens)
		authGroup.GET("/tokens/:id", GetTokens)
		authGroup.DELETE("/tokens/:id", DeleteToken)
	}
}

//GenerateToken generates token from context
func GenerateToken(c *gin.Context) {
	var currentUser *User

	if accessToken, ok := c.GetQuery("access_token"); ok {
		githubUser, err := GetGithubUser(accessToken)
		if err != nil {
			err := c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("Invalid session"))
			log.Info(c.ClientIP(), " ", err.Error())
			return
		}
		user := User{}
		err = Auth.GetDB(c.Request).
			Joins("left join auth_identities on users.id = auth_identities.user_id").
			Where("auth_identities.uid = ?", githubUser.GetID()).
			Find(&user).Error
		if err != nil {
			err := c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("Invalid session"))
			log.Info(c.ClientIP(), " ", err.Error())
			return
		}
		currentUser = &user
	} else {
		Handler(c)
		if c.IsAborted() {
			return
		}
		currentUser = GetCurrentUser(c.Request)
		if currentUser == nil {
			err := c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("Invalid session"))
			log.Info(c.ClientIP(), " ", err.Error())
			return
		}
	}

	tokenRequest := struct {
		Name        string `json:"name,omitempty"`
		VirtualUser string `json:"virtualUser,omitempty"`
	}{Name: "generated"}

	if c.Request.Method == http.MethodPost && c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&tokenRequest); err != nil {
			err := c.AbortWithError(http.StatusBadRequest, err)
			log.Info(c.ClientIP(), " ", err.Error())
			return
		}
	}

	isForVirtualUser := tokenRequest.VirtualUser != ""

	userID := currentUser.IDString()
	userLogin := currentUser.Login
	tokenType := DroneUserTokenType
	if isForVirtualUser {
		userID = tokenRequest.VirtualUser
		userLogin = tokenRequest.VirtualUser
		tokenType = DroneHookTokenType
	}

	tokenID, signedToken, err := createAndStoreAPIToken(userID, userLogin, tokenType, tokenRequest.Name)

	if err != nil {
		err = c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("%s", err))
		log.Info(c.ClientIP(), " ", err.Error())
		return
	}

	if isForVirtualUser {
		orgName := GetOrgNameFromVirtualUser(tokenRequest.VirtualUser)
		organization := Organization{Name: orgName}
		err = Auth.GetDB(c.Request).
			Model(currentUser).
			Where(&organization).
			Related(&organization, "Organizations").Error
		if err != nil {
			statusCode := GormErrorToStatusCode(err)
			err = c.AbortWithError(statusCode, err)
			log.Info(c.ClientIP(), " ", err.Error())
			return
		}

		AddDefaultRoleForVirtualUser(userID)
		AddOrgRoleForUser(userID, organization.ID)
	}

	c.JSON(http.StatusOK, gin.H{"id": tokenID, "token": signedToken})
}

func createAPIToken(userID string, userLogin string, tokenType bauth.TokenType) (string, string, error) {
	tokenID := uuid.NewV4().String()

	// Create the Claims
	claims := &bauth.ScopedClaims{
		StandardClaims: jwt.StandardClaims{
			Issuer:    JwtIssuer,
			Audience:  JwtAudience,
			IssuedAt:  jwt.TimeFunc().Unix(),
			ExpiresAt: 0,
			Subject:   userID,
			Id:        tokenID,
		},
		Scope: "api:invoke", // "scope" for Pipeline
		Type:  tokenType,    // "type" for Drone
		Text:  userLogin,    // "text" for Drone
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := jwtToken.SignedString([]byte(signingKeyBase32))
	if err != nil {
		return "", "", errors.Wrap(err, "Failed to sign user token")
	}

	return tokenID, signedToken, nil
}

func createAndStoreAPIToken(userID string, userLogin string, tokenType bauth.TokenType, tokenName string) (string, string, error) {
	tokenID, signedToken, err := createAPIToken(userID, userLogin, tokenType)
	if err != nil {
		return "", "", err
	}

	token := bauth.NewToken(tokenID, tokenName)
	err = TokenStore.Store(userID, token)
	if err != nil {
		return "", "", errors.Wrap(err, "Failed to store user token")
	}

	return tokenID, signedToken, nil
}

// GetTokens returns the calling user's access tokens
func GetTokens(c *gin.Context) {
	currentUser := GetCurrentUser(c.Request)
	if currentUser == nil {
		err := c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("Invalid session"))
		log.Info(c.ClientIP(), " ", err.Error())
		return
	}
	tokenID := c.Param("id")

	if tokenID == "" {
		tokens, err := TokenStore.List(currentUser.IDString())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		} else {
			c.JSON(http.StatusOK, tokens)
		}
	} else {
		token, err := TokenStore.Lookup(currentUser.IDString(), tokenID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		} else if token != nil {
			c.JSON(http.StatusOK, token)
		} else {
			c.AbortWithStatusJSON(http.StatusNotFound, pkgCommon.ErrorResponse{
				Code:    http.StatusNotFound,
				Message: "Token not found",
				Error:   "Token not found",
			})
		}
	}
}

// DeleteToken deletes the calling user's access token specified by token id
func DeleteToken(c *gin.Context) {
	currentUser := GetCurrentUser(c.Request)
	if currentUser == nil {
		err := c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("Invalid session"))
		log.Info(c.ClientIP(), err.Error())
		return
	}
	tokenID := c.Param("id")

	if tokenID == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, fmt.Errorf("Missing token id"))
	} else {
		err := TokenStore.Revoke(currentUser.IDString(), tokenID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		} else {
			c.Status(http.StatusNoContent)
		}
	}
}

//BanzaiSessionStorer stores the banzai session
type BanzaiSessionStorer struct {
	auth.SessionStorer
}

//Update updates the BanzaiSessionStorer
func (sessionStorer *BanzaiSessionStorer) Update(w http.ResponseWriter, req *http.Request, claims *claims.Claims) error {
	token := sessionStorer.SignedToken(claims)
	err := sessionStorer.SessionManager.Add(w, req, sessionStorer.SessionName, token)
	if err != nil {
		log.Info(req.RemoteAddr, err.Error())
		return err
	}

	// Set the drone cookie as well, but that cookie's value is actually a Pipeline API token
	currentUser := GetCurrentUser(req)
	if currentUser == nil {
		return fmt.Errorf("Can't get current user")
	}

	// Drone tokens have to stored in Vault, because they act as Pipeline API tokens as well
	// TODO We need GC them somehow
	_, droneToken, err := createAndStoreAPIToken(claims.UserID, currentUser.Login, DroneUserTokenType, "Drone session token")
	if err != nil {
		log.Info(req.RemoteAddr, err.Error())
		return err
	}
	SetCookie(w, req, DroneSessionCookie, droneToken)
	return nil
}

// BanzaiLogoutHandler does the qor/auth DefaultLogoutHandler default logout behavior + deleting the Drone cookie
func BanzaiLogoutHandler(context *auth.Context) {
	DelCookie(context.Writer, context.Request, DroneSessionCookie)
	DelCookie(context.Writer, context.Request, PipelineSessionCookie)
	auth.DefaultLogoutHandler(context)
}

// BanzaiDeregisterHandler deletes the user and all his/her tokens from the database
func BanzaiDeregisterHandler(context *auth.Context) {
	user, err := GetCurrentUserFromDB(context.Request)
	if user == nil {
		http.Error(context.Writer, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(context.Writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	db := context.GetDB(context.Request)

	userAdminOrganizations := []UserOrganization{}

	// Query the organizations where the only admin is the current user.
	sql :=
		`SELECT * FROM user_organizations WHERE role = ? AND organization_id IN
		(SELECT DISTINCT organization_id FROM user_organizations WHERE user_id = ? AND role = ?)
		GROUP BY user_id, organization_id
		HAVING COUNT(*) = 1`

	if err := db.Raw(sql, "admin", user.ID, "admin").Scan(&userAdminOrganizations).Error; err != nil {
		log.Errorln("Failed select user only owned organizations:", err)
		http.Error(context.Writer, err.Error(), http.StatusInternalServerError)
		return
	}

	// If there are any organizations with only this user as admin, throw an error
	if len(userAdminOrganizations) != 0 {
		orgs := []string{}
		for _, org := range userAdminOrganizations {
			orgs = append(orgs, fmt.Sprint(org.OrganizationID))
		}
		http.Error(context.Writer, "You must remove yourself or transfer ownership or delete these organizations before you can delete your user: "+strings.Join(orgs, ", "), http.StatusBadRequest)
		return
	}

	if err := db.Model(user).Association("Organizations").Clear().Error; err != nil {
		log.Errorln("Failed delete user's organization associations:", err)
		http.Error(context.Writer, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := db.Delete(user).Error; err != nil {
		log.Errorln("Failed delete user from DB:", err)
		http.Error(context.Writer, err.Error(), http.StatusInternalServerError)
		return
	}

	authIdentity := &AuthIdentity{Basic: auth_identity.Basic{UserID: user.IDString()}}
	if err := db.Delete(authIdentity).Error; err != nil {
		log.Errorln("Failed delete user's auth_identity from DB:", err)
		http.Error(context.Writer, err.Error(), http.StatusInternalServerError)
		return
	}

	droneUser := DroneUser{Login: user.Login}
	// We need to pass droneUser as well as the where clause, because Delete() filters by primary
	// key by default: http://doc.gorm.io/crud.html#delete but here we need to delete by the Login
	if err := DroneDB.Delete(droneUser, droneUser).Error; err != nil {
		log.Errorln("Failed delete user from Drone:", err)
		http.Error(context.Writer, err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete Tokens
	tokens, err := TokenStore.List(user.IDString())
	if err != nil {
		log.Errorln("Failed list user's tokens during user deletetion:", err)
		http.Error(context.Writer, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, token := range tokens {
		err = TokenStore.Revoke(user.IDString(), token.ID)
		if err != nil {
			log.Errorln("Failed remove user's tokens during user deletetion:", err)
			http.Error(context.Writer, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Delete Casbin roles
	DeleteRolesForUser(user.ID)

	BanzaiLogoutHandler(context)
}

// GetOrgNameFromVirtualUser returns the organization name for which the virtual user has access
func GetOrgNameFromVirtualUser(virtualUser string) string {
	return strings.Split(virtualUser, "/")[0]
}
