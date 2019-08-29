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
	"strings"
	"time"

	"emperror.dev/emperror"
	bauth "github.com/banzaicloud/bank-vaults/pkg/sdk/auth"
	"github.com/banzaicloud/cicd-go/cicd"
	"github.com/dgrijalva/jwt-go"
	"github.com/jinzhu/copier"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/qor/auth"
	"github.com/qor/auth/auth_identity"
	"github.com/qor/qor/utils"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"

	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/helm"
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

const (
	RoleAdmin  = "admin"
	RoleMember = "member"
)

// nolint: gochecknoglobals
var roleLevelMap = map[string]int{
	RoleAdmin:  100,
	RoleMember: 50,
}

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

// UserOrganization describes the user organization
type UserOrganization struct {
	UserID         uint
	OrganizationID uint
	Role           string `gorm:"default:'member'"`
}

// Organization struct
type Organization struct {
	ID        uint      `gorm:"primary_key" json:"id"`
	GithubID  *int64    `gorm:"unique" json:"githubId,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Name      string    `gorm:"unique;not null" json:"name"`
	Provider  string    `gorm:"not null" json:"provider"`
	Users     []User    `gorm:"many2many:user_organizations" json:"users,omitempty"`
	Role      string    `json:"-" gorm:"-"` // Used only internally
}

// IDString returns the ID as string
func (user *User) IDString() string {
	return fmt.Sprint(user.ID)
}

// IDString returns the ID as string
func (org *Organization) IDString() string {
	return fmt.Sprint(org.ID)
}

// TableName sets CICDUser's table name
func (CICDUser) TableName() string {
	return "users"
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
	claims := &CICDClaims{Type: CICDUserTokenType, Text: login}
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
	cicdDB           *gorm.DB
	events           authEvents
	orgImporter      *OrgImporter
}

func getOrganizationsFromIDToken(idTokenClaims *IDTokenClaims) (map[string][]string, error) {
	organizations := make(map[string][]string)

	for _, group := range idTokenClaims.Groups {
		// get the part before :, that will be the organization name
		s := strings.SplitN(group, ":", 2)
		if len(s) < 1 {
			return nil, errors.New("invalid group")
		}

		if _, ok := organizations[s[0]]; !ok {
			organizations[s[0]] = make([]string, 0)
		}

		if len(s) > 1 && s[1] != "" {
			organizations[s[0]] = append(organizations[s[0]], s[1])
		}
	}

	return organizations, nil
}

func getOrganizationsFromSchema(schema *auth.Schema) (map[string][]string, error) {
	idTokenClaims := schema.RawInfo.(*IDTokenClaims)
	return getOrganizationsFromIDToken(idTokenClaims)
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

	organizations, err := getOrganizationsFromSchema(schema)
	if err != nil {
		return nil, "", emperror.Wrap(err, "failed to parse groups/organizations")
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

	db := authCtx.Auth.GetDB(authCtx.Request)

	// TODO we should call the Drone API instead and insert the token later on manually by the user
	if schema.Provider == ProviderDexGithub || schema.Provider == ProviderDexGitlab {
		err = bus.createUserInCICDDB(currentUser)
		if err != nil {
			return nil, "", emperror.Wrap(err, "failed to create user in CICD database")
		}
	}

	// When a user registers a default organization is created in which he/she is admin
	userOrg := Organization{
		Name:     currentUser.Login,
		Provider: getBackendProvider(schema.Provider),
	}
	currentUser.Organizations = []Organization{userOrg}

	err = db.Create(currentUser).Error
	if err != nil {
		return nil, "", emperror.Wrap(err, "failed to create user organization")
	}

	err = helm.InstallLocalHelm(helm.GenerateHelmRepoEnv(currentUser.Organizations[0].Name))
	if err != nil {
		log.Errorf("Error during local helm install: %s", err.Error())
	}

	bus.events.OrganizationRegistered(currentUser.Organizations[0].ID, currentUser.ID)

	// Import organizations in case of DEX
	err = bus.orgImporter.ImportOrganizationsFromDex(currentUser, organizations, getBackendProvider(schema.Provider))

	return currentUser, fmt.Sprint(db.NewScope(currentUser).PrimaryKeyValue()), err
}

// SaveUserSCMToken saves a personal access token specified for a user
func SaveUserSCMToken(user *User, scmToken string, tokenType string) error {
	// Revoke the old Github token from Vault if any
	err := TokenStore.Revoke(user.IDString(), tokenType)
	if err != nil {
		return errors.Wrap(err, "failed to revoke old access token")
	}
	token := bauth.NewToken(tokenType, "scm access token")
	token.Value = scmToken
	err = TokenStore.Store(user.IDString(), token)
	if err != nil {
		return emperror.WrapWith(err, "failed to store access token for user", "user", user.Login)
	}
	if tokenType == GithubTokenID || tokenType == GitlabTokenID {
		// TODO CICD should use Vault as well, and this should be removed by then
		err = updateUserInCICDDB(user, scmToken)
		if err != nil {
			return emperror.WrapWith(err, "failed to update access token for user in CICD", "user", user.Login)
		}

		synchronizeCICDRepos(user.Login)
	}

	return nil
}

// RemoveUserSCMToken removes a GitHub personal access token specified for a user
func RemoveUserSCMToken(user *User, tokenType string) error {
	// Revoke the old Github token from Vault if any
	err := TokenStore.Revoke(user.IDString(), tokenType)
	if err != nil {
		return errors.Wrap(err, "failed to revoke access token")
	}

	if tokenType == GithubTokenID || tokenType == GitlabTokenID {
		// TODO CICD should use Vault as well, and this should be removed by then
		err = updateUserInCICDDB(user, "")
		if err != nil {
			return emperror.WrapWith(err, "failed to revoke access token for user in CICD", "user", user.Login)
		}
	}
	return nil
}

func GetOAuthRefreshToken(userID string) (string, error) {
	token, err := TokenStore.Lookup(userID, OAuthRefreshTokenID)
	if err != nil {
		return "", emperror.Wrap(err, "failed to lookup user refresh token")
	}

	if token == nil {
		return "", nil
	}

	return token.Value, nil
}

func SaveOAuthRefreshToken(userID string, refreshToken string) error {
	// Revoke the old refresh token from Vault if any
	err := TokenStore.Revoke(userID, OAuthRefreshTokenID)
	if err != nil {
		return errors.Wrap(err, "failed to revoke old refresh token")
	}
	token := bauth.NewToken(OAuthRefreshTokenID, "OAuth refresh token")
	token.Value = refreshToken
	err = TokenStore.Store(userID, token)
	if err != nil {
		return emperror.WrapWith(err, "failed to store refresg token for user", "user", userID)
	}

	return nil
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

// OrgImporter imports organizations.
type OrgImporter struct {
	db     *gorm.DB
	events authEvents
}

// NewOrgImporter returns a new OrgImporter instance.
func NewOrgImporter(
	db *gorm.DB,
	events eventBus,
) *OrgImporter {
	return &OrgImporter{
		db:     db,
		events: ebAuthEvents{eb: events},
	}
}

func (i *OrgImporter) ImportOrganizationsFromGithub(currentUser *User, githubToken string) error {
	orgs, err := getGithubOrganizations(githubToken)
	if err != nil {
		return emperror.Wrap(err, "failed to get organizations")
	}

	return i.ImportOrganizations(currentUser, orgs)
}

func (i *OrgImporter) ImportOrganizationsFromGitlab(currentUser *User, gitlabToken string) error {
	orgs, err := getGitlabOrganizations(gitlabToken)
	if err != nil {
		return emperror.Wrap(err, "failed to get organizations")
	}

	return i.ImportOrganizations(currentUser, orgs)
}

func (i *OrgImporter) ImportOrganizationsFromDex(currentUser *User, organizations map[string][]string, provider string) error {
	var orgs []organization
	for org, groups := range organizations {
		role := RoleMember

		if provider == ProviderGithub || provider == ProviderGitlab {
			role = RoleAdmin
		} else {
			// TODO: add role group binding
			for _, group := range groups {
				if roleLevelMap[role] < roleLevelMap[group] {
					role = group
				}
			}
		}

		orgs = append(orgs, organization{name: org, provider: provider, role: role})
	}

	return i.ImportOrganizations(currentUser, orgs)
}

func (i *OrgImporter) ImportOrganizations(currentUser *User, orgs []organization) error {
	orgIDs, err := importOrganizations(i.db, currentUser, orgs)

	if err != nil {
		return emperror.Wrap(err, "failed to import organizations")
	}

	for id, created := range orgIDs {
		if created {
			i.events.OrganizationRegistered(id, currentUser.ID)
		}
	}

	return nil
}

func importOrganizations(db *gorm.DB, currentUser *User, orgs []organization) (map[uint]bool, error) {
	orgIDs := make(map[uint]bool, len(orgs))

	tx := db.Begin()
	for _, org := range orgs {
		o := Organization{
			Name:     org.name,
			Role:     org.role,
			Provider: org.provider,
		}

		needsCreation := true
		err := tx.Where(o).First(&o).Error
		if err == nil {
			orgIDs[o.ID] = false
			needsCreation = false

			// We don't need to create the organization again imported from the provider
			// however we need to associate the user with this organization already in db

		} else if !gorm.IsRecordNotFoundError(err) {
			tx.Rollback()

			return nil, errors.Wrap(err, "failed to check if organization exists")
		}

		if needsCreation {
			err = tx.Where(o).Create(&o).Error
			if err != nil {
				tx.Rollback()

				return nil, errors.Wrap(err, "failed to create organization")
			}

			orgIDs[o.ID] = true
		}

		err = tx.Model(currentUser).Association("Organizations").Append(o).Error
		if err != nil {
			tx.Rollback()

			return nil, errors.Wrap(err, "failed to associate user with organization")
		}

		userRoleInOrg := UserOrganization{UserID: currentUser.ID, OrganizationID: o.ID}
		err = tx.Model(&UserOrganization{}).Where(userRoleInOrg).Update("role", o.Role).Error
		if err != nil {
			tx.Rollback()

			return nil, errors.Wrap(err, "failed to save user role in organization")
		}
	}

	err := tx.Commit().Error
	if err != nil {
		return nil, errors.Wrap(err, "failed to save organizations")
	}

	return orgIDs, nil
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
