// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"emperror.dev/emperror"
	"github.com/banzaicloud/cicd-go/cicd"
	ginauth "github.com/banzaicloud/gin-utilz/auth"
	"github.com/dgrijalva/jwt-go"
	"github.com/jinzhu/copier"
	"github.com/jinzhu/gorm"
	"github.com/qor/auth"
	"github.com/qor/auth/auth_identity"
	"github.com/qor/qor/utils"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"

	"github.com/banzaicloud/pipeline/config"
)

const (
	// CurrentOrganization current organization key
	CurrentOrganization utils.ContextKey = "org"

	currentOrganizationID utils.ContextKey = "orgID"

	// SignUp is present if the current request is a signing up
	SignUp utils.ContextKey = "signUp"

	// GithubTokenID denotes the tokenID for the user's Github token, there can be only one
	GithubTokenID = "github"
	// GitlabTokenID denotes the tokenID for the user's Github token, there can be only one
	GitlabTokenID = "gitlab"

	// OAuthRefreshTokenID denotes the tokenID for the user's OAuth refresh token, there can be only one
	OAuthRefreshTokenID = "oauth_refresh"
)

// AuthIdentity auth identity session model
type AuthIdentity struct {
	ID        uint      `gorm:"primary_key" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	auth_identity.Basic
	auth_identity.SignLogs
}

// User struct
type User struct {
	ID            uint           `gorm:"primary_key" json:"id"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	Name          string         `form:"name" json:"name,omitempty"`
	Email         string         `form:"email" json:"email,omitempty"`
	Login         string         `gorm:"unique;not null" form:"login" json:"login"`
	Image         string         `form:"image" json:"image,omitempty"`
	Organizations []Organization `gorm:"many2many:user_organizations" json:"organizations,omitempty"`
	Virtual       bool           `json:"-" gorm:"-"` // Used only internally
	APIToken      string         `json:"-" gorm:"-"` // Used only internally
}

// CICDUser struct
type CICDUser struct {
	ID     int64  `gorm:"column:user_id;primary_key"`
	Login  string `gorm:"column:user_login"`
	Token  string `gorm:"column:user_token"`
	Secret string `gorm:"column:user_secret"`
	Expiry int64  `gorm:"column:user_expiry"`
	Email  string `gorm:"column:user_email"`
	Image  string `gorm:"column:user_avatar"`
	Active bool   `gorm:"column:user_active"`
	Admin  bool   `gorm:"column:user_admin"`
	Hash   string `gorm:"column:user_hash"`
	Synced int64  `gorm:"column:user_synced"`
}

// UserOrganization describes a user organization membership.
type UserOrganization struct {
	User   User
	UserID uint

	Organization   Organization
	OrganizationID uint

	Role string `gorm:"default:'member'"`
}

// IDString returns the ID as string
func (user *User) IDString() string {
	return fmt.Sprint(user.ID)
}

// TableName sets CICDUser's table name
func (CICDUser) TableName() string {
	return "users"
}

type UserExtractor struct{}

func (e UserExtractor) GetUserID(ctx context.Context) (uint, bool) {
	if user, ok := ctx.Value(auth.CurrentUser).(*User); ok {
		return user.ID, true
	}

	return 0, false
}

func (e UserExtractor) GetUserLogin(ctx context.Context) (string, bool) {
	if user, ok := ctx.Value(auth.CurrentUser).(*User); ok {
		return user.Login, true
	}

	return "", false
}

// GetCurrentUser returns the current user
func GetCurrentUser(req *http.Request) *User {
	if currentUser, ok := Auth.GetCurrentUser(req).(*User); ok {
		if currentUser != nil && currentUser.APIToken == "" {
			apiToken, _ := parseRawTokenFromRequest(req)
			currentUser.APIToken = apiToken
		}
		return currentUser
	}
	return nil
}

// GetCurrentOrganization return the user's organization
func GetCurrentOrganization(req *http.Request) *Organization {
	if organization := req.Context().Value(CurrentOrganization); organization != nil {
		return organization.(*Organization)
	}
	return nil
}

// GetCurrentOrganizationID return the user's organization ID.
func GetCurrentOrganizationID(ctx context.Context) (uint, bool) {
	if orgID, ok := ctx.Value(currentOrganizationID).(uint); ok {
		return orgID, true
	}
	if organization := ctx.Value(CurrentOrganization); organization != nil {
		return organization.(*Organization).ID, true
	}

	return 0, false
}

// SetCurrentOrganizationID returns a context with the organization ID set
func SetCurrentOrganizationID(ctx context.Context, orgID uint) context.Context {
	return context.WithValue(ctx, currentOrganizationID, orgID)
}

// NewCICDClient creates an authenticated CICD client for the user specified by the JWT in the HTTP request
func NewCICDClient(apiToken string) cicd.Client {
	cicdURL := viper.GetString("cicd.url")
	config := new(oauth2.Config)
	httpClient := http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: viper.GetBool("cicd.insecure"),
			},
		},
	}
	ctx := context.Background()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, &httpClient)
	client := config.Client(
		ctx,
		&oauth2.Token{
			AccessToken: apiToken,
		},
	)
	return cicd.NewClient(cicdURL, client)
}

// NewTemporaryCICDClient creates an authenticated CICD client for the user specified by login
func NewTemporaryCICDClient(login string) (cicd.Client, error) {
	// Create a temporary CICD API token
	claims := &CICDClaims{Type: ginauth.TokenType(CICDUserTokenType), Text: login}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	cicdAPIToken, err := token.SignedString([]byte(signingKeyBase32))
	if err != nil {
		log.Errorln("Failed to create temporary CICD token", err.Error())
		return nil, err
	}

	return NewCICDClient(cicdAPIToken), nil
}

// BanzaiUserStorer struct
type BanzaiUserStorer struct {
	auth.UserStorer
	signingKeyBase32 string // CICD uses base32 Hash
	db               *gorm.DB
	cicdDB           *gorm.DB
	orgSyncer        OIDCOrganizationSyncer
}

func emailToLoginName(email string) string {
	filterRegexp := regexp.MustCompile("[^a-zA-Z0-9-]+")
	replaceRegexp := regexp.MustCompile("[@.]+")

	login := replaceRegexp.ReplaceAllString(email, "-")
	login = filterRegexp.ReplaceAllString(login, "")

	return login
}

// Save differs from the default UserStorer.Save() in that it
// extracts Token and Login and saves to CICD DB as well
func (bus BanzaiUserStorer) Save(schema *auth.Schema, authCtx *auth.Context) (user interface{}, userID string, err error) {
	currentUser := &User{}
	err = copier.Copy(currentUser, schema)
	if err != nil {
		return nil, "", err
	}

	// Until https://github.com/dexidp/dex/issues/1076 gets resolved we need to use a manual
	// GitHub API query to get the user login and image to retain compatibility for now

	switch schema.Provider {
	case ProviderDexGithub:
		githubUserMeta, err := getGithubUserMeta(schema)
		if err != nil {
			return nil, "", emperror.Wrap(err, "failed to query github login name")
		}
		currentUser.Login = githubUserMeta.Login
		currentUser.Image = githubUserMeta.AvatarURL

	case ProviderDexGitlab:

		gitlabUserMeta, err := getGitlabUserMeta(schema)
		if err != nil {
			return nil, "", emperror.Wrap(err, "failed to query gitlab login name")
		}
		currentUser.Login = gitlabUserMeta.Username
		currentUser.Image = gitlabUserMeta.AvatarURL

	default:
		// Login will be derived from the email for new users coming from an other provider than GitHub and GitLab
		currentUser.Login = emailToLoginName(schema.Email)
	}

	// TODO we should call the Drone API instead and insert the token later on manually by the user
	if viper.GetBool("cicd.enabled") && (schema.Provider == ProviderDexGithub || schema.Provider == ProviderDexGitlab) {
		err = bus.createUserInCICDDB(currentUser)
		if err != nil {
			return nil, "", emperror.Wrap(err, "failed to create user in CICD database")
		}
	}

	err = bus.db.Create(currentUser).Error
	if err != nil {
		return nil, "", emperror.Wrap(err, "failed to create user organization")
	}

	err = bus.orgSyncer.SyncOrganizations(authCtx.Request.Context(), *currentUser, schema.RawInfo.(*IDTokenClaims))

	return currentUser, fmt.Sprint(bus.db.NewScope(currentUser).PrimaryKeyValue()), err
}

// Update updates the user's group mmeberships from the OIDC ID token at every login
func (bus BanzaiUserStorer) Update(schema *auth.Schema, authCtx *auth.Context) (err error) {
	currentUser := User{}

	err = bus.db.Where("id = ?", schema.UID).First(&currentUser).Error
	if err != nil {
		return err
	}

	return bus.orgSyncer.SyncOrganizations(authCtx.Request.Context(), currentUser, schema.RawInfo.(*IDTokenClaims))
}

func (bus BanzaiUserStorer) createUserInCICDDB(user *User) error {
	cicdUser := &CICDUser{
		Login:  user.Login,
		Email:  user.Email,
		Hash:   bus.signingKeyBase32,
		Image:  user.Image,
		Active: true,
		Admin:  true,
		Synced: time.Now().Unix(),
	}
	return bus.cicdDB.Where(cicdUser).FirstOrCreate(cicdUser).Error
}

func updateUserInCICDDB(user *User, scmAccessToken string) error {
	where := &CICDUser{
		Login: user.Login,
	}
	update := map[string]interface{}{
		"user_token":  scmAccessToken,
		"user_synced": time.Now().Unix(),
	}
	return cicdDB.Model(&CICDUser{}).Where(where).Update(update).Error
}

// This method tries to call the CICD API on a best effort basis to fetch all repos before the user navigates there.
func synchronizeCICDRepos(login string) {
	cicdClient, err := NewTemporaryCICDClient(login)
	if err != nil {
		log.Warnln("failed to create CICD client", err.Error())
	}
	_, err = cicdClient.RepoListOpts(true, true)
	if err != nil {
		log.Warnln("failed to sync CICD repositories", err.Error())
	}
}

// GetOrganizationById returns an organization from database by ID
func GetOrganizationById(orgID uint) (*Organization, error) {
	db := config.DB()
	var org Organization
	err := db.Find(&org, Organization{ID: orgID}).Error
	return &org, err
}

// GetOrganizationByName returns an organization from database by Name
func GetOrganizationByName(name string) (*Organization, error) {
	db := config.DB()
	var org Organization
	err := db.Find(&org, Organization{Name: name}).Error
	return &org, err
}

// GetUserById returns user
func GetUserById(userId uint) (*User, error) {
	db := config.DB()
	var user User
	err := db.Find(&user, User{ID: userId}).Error
	return &user, err
}

// GetUserByLoginName returns user
func GetUserByLoginName(login string) (*User, error) {
	db := config.DB()
	var user User
	err := db.Find(&user, User{Login: login}).Error
	return &user, err
}

// GetUserNickNameById returns user's login name
func GetUserNickNameById(userId uint) (userName string) {
	if userId == 0 {
		return
	}

	if user, err := GetUserById(userId); err != nil {
		log.Warnf("Error during getting user name: %s", err.Error())
	} else {
		userName = user.Login
	}

	return
}

func parseRawTokenFromRequest(r *http.Request) (string, error) {
	var token = r.Header.Get("Authorization")

	// first we attempt to get the token from the
	// authorization header.
	if len(token) != 0 {
		token = r.Header.Get("Authorization")
		_, err := fmt.Sscanf(token, "Bearer %s", &token)
		return token, err
	}

	// then we attempt to get the token from the
	// access_token url query parameter
	token = r.FormValue("access_token")
	if len(token) != 0 {
		return token, nil
	}

	// and finally we attempt to get the token from
	// the user session cookie
	cookie, err := r.Cookie("user_sess")
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}
