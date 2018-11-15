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
	"database/sql/driver"
	"fmt"
	"net/http"
	"time"

	bauth "github.com/banzaicloud/bank-vaults/pkg/auth"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/dgrijalva/jwt-go"
	"github.com/drone/drone-go/drone"
	"github.com/jinzhu/copier"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql" // blank import is used here for sql driver inclusion
	"github.com/pkg/errors"
	"github.com/qor/auth"
	"github.com/qor/auth/auth_identity"
	"github.com/qor/qor/utils"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

const (
	// CurrentOrganization current organization key
	CurrentOrganization utils.ContextKey = "org"

	// GithubTokenID denotes the tokenID for the user's Github token, there can be only one
	GithubTokenID = "github"
)

// AuthIdentity auth identity session model
type AuthIdentity struct {
	ID        uint      `gorm:"primary_key" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	auth_identity.Basic
	auth_identity.SignLogs
}

// WhitelistedAuthIdentity auth identity whitelist session model
type WhitelistedAuthIdentity struct {
	ID        uint         `gorm:"primary_key" json:"id"`
	CreatedAt time.Time    `json:"createdAt"`
	UpdatedAt time.Time    `json:"updatedAt"`
	Provider  string       `gorm:"unique_index:provider_login"` // phone, email, github, google...
	Type      IdentityType `gorm:"type:ENUM('User', 'Organization')"`
	Login     string       `gorm:"unique_index:provider_login"`
	UID       string       `gorm:"column:uid"`
}

func (WhitelistedAuthIdentity) TableName() string {
	return "whitelisted_auth_identities"
}

type IdentityType string

const (
	UserType         IdentityType = "User"
	OrganizationType IdentityType = "Organization"
)

func (t *IdentityType) Scan(value interface{}) error { *t = IdentityType(value.([]byte)); return nil }
func (t IdentityType) Value() (driver.Value, error)  { return string(t), nil }

//User struct
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
}

//DroneUser struct
type DroneUser struct {
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
	Role           string `gorm:"default:'admin'"`
}

//Organization struct
type Organization struct {
	ID        uint      `gorm:"primary_key" json:"id"`
	GithubID  *int64    `gorm:"unique" json:"githubId,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Name      string    `gorm:"unique;not null" json:"name"`
	Users     []User    `gorm:"many2many:user_organizations" json:"users,omitempty"`
	Role      string    `json:"-" gorm:"-"` // Used only internally
}

//IDString returns the ID as string
func (user *User) IDString() string {
	return fmt.Sprint(user.ID)
}

//IDString returns the ID as string
func (org *Organization) IDString() string {
	return fmt.Sprint(org.ID)
}

//TableName sets DroneUser's table name
func (DroneUser) TableName() string {
	return "users"
}

// GetCurrentUser returns the current user
func GetCurrentUser(req *http.Request) *User {
	if currentUser, ok := Auth.GetCurrentUser(req).(*User); ok {
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

func newDroneClient(apiToken string) drone.Client {
	droneURL := viper.GetString("drone.url")
	config := new(oauth2.Config)
	client := config.Client(
		context.Background(),
		&oauth2.Token{
			AccessToken: apiToken,
		},
	)
	return drone.NewClient(droneURL, client)
}

// NewTemporaryDroneClient creates an authenticated Drone client for the user specified by login
func NewTemporaryDroneClient(login string) (drone.Client, error) {
	// Create a temporary Drone API token
	claims := &DroneClaims{Type: DroneUserTokenType, Text: login}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	droneAPIToken, err := token.SignedString([]byte(signingKeyBase32))
	if err != nil {
		log.Errorln("Failed to create temporary Drone token", err.Error())
		return nil, err
	}

	return newDroneClient(droneAPIToken), nil
}

// NewDroneClient creates an authenticated Drone client for the user specified by the JWT in the HTTP request
func NewDroneClient(request *http.Request) (drone.Client, error) {
	droneAPIToken, err := parseDroneTokenFromRequest(request)
	if err != nil {
		log.Errorln("Failed to parse Drone token", err.Error())
		return nil, err
	}

	return newDroneClient(droneAPIToken), nil
}

//BanzaiUserStorer struct
type BanzaiUserStorer struct {
	auth.UserStorer
	signingKeyBase32 string // Drone uses base32 Hash
	droneDB          *gorm.DB
	events           authEvents
	accessManager    accessManager
}

// Save differs from the default UserStorer.Save() in that it
// extracts Token and Login and saves to Drone DB as well
func (bus BanzaiUserStorer) Save(schema *auth.Schema, context *auth.Context) (user interface{}, userID string, err error) {

	currentUser := &User{}
	err = copier.Copy(currentUser, schema)
	if err != nil {
		return nil, "", err
	}

	// This assumes GitHub auth only right now
	githubExtraInfo := schema.RawInfo.(*GithubExtraInfo)
	currentUser.Login = githubExtraInfo.Login
	err = bus.createUserInDroneDB(currentUser, githubExtraInfo.Token)
	if err != nil {
		log.Info(context.Request.RemoteAddr, err.Error())
		return nil, "", err
	}

	synchronizeDroneRepos(currentUser.Login)

	// When a user registers a default organization is created in which he/she is admin
	userOrg := Organization{
		Name: currentUser.Login,
	}
	currentUser.Organizations = []Organization{userOrg}

	db := context.Auth.GetDB(context.Request)
	err = db.Create(currentUser).Error
	if err != nil {
		return nil, "", fmt.Errorf("failed to create user organization: %s", err.Error())
	}

	err = helm.InstallLocalHelm(helm.GenerateHelmRepoEnv(currentUser.Organizations[0].Name))
	if err != nil {
		log.Errorf("Error during local helm install: %s", err.Error())
	}

	bus.accessManager.GrantDefaultAccessToUser(currentUser.IDString())

	// Save the Github token to Vault
	token := bauth.NewToken(GithubTokenID, "Github access token")
	token.Value = githubExtraInfo.Token
	err = TokenStore.Store(fmt.Sprint(currentUser.ID), token)
	if err != nil {
		return "", "", fmt.Errorf("failed to store Github access token: %s", err.Error())
	}

	bus.accessManager.AddOrganizationPolicies(currentUser.Organizations[0].ID)
	bus.accessManager.GrantOganizationAccessToUser(currentUser.IDString(), currentUser.Organizations[0].ID)
	bus.events.OrganizationRegistered(currentUser.Organizations[0].ID)

	githubOrgIDs, err := importGithubOrganizations(db, currentUser, githubExtraInfo.Token)

	if err == nil {
		for id, created := range githubOrgIDs {
			bus.accessManager.AddOrganizationPolicies(id)
			bus.accessManager.GrantOganizationAccessToUser(currentUser.IDString(), id)

			if created {
				bus.events.OrganizationRegistered(id)
			}
		}
	}

	return currentUser, fmt.Sprint(db.NewScope(currentUser).PrimaryKeyValue()), err
}

// Update differs from the default UserStorer.Update() in that it
// updates the GitHub access token of the given user
func (bus BanzaiUserStorer) Update(schema *auth.Schema, context *auth.Context) error {

	// This assumes GitHub auth only right now
	currentUser := &User{}
	githubExtraInfo := schema.RawInfo.(*GithubExtraInfo)
	currentUser.Login = githubExtraInfo.Login

	// Revoke the old Github token from Vault
	err := TokenStore.Revoke(context.Claims.UserID, GithubTokenID)
	if err != nil {
		return errors.Wrap(err, "failed to revoke old Github access token")
	}

	// Save the new Github token to Vault
	token := bauth.NewToken(GithubTokenID, "Github access token")
	token.Value = githubExtraInfo.Token
	err = TokenStore.Store(context.Claims.UserID, token)
	if err != nil {
		return errors.Wrap(err, "failed to save Github access token")
	}

	// Also update the new Github token in Drone (TODO Drone should get it from Vault as well)
	return bus.updateUserInDroneDB(currentUser, githubExtraInfo.Token)
}

func (bus BanzaiUserStorer) createUserInDroneDB(user *User, githubAccessToken string) error {
	droneUser := &DroneUser{
		Login:  user.Login,
		Email:  user.Email,
		Token:  githubAccessToken,
		Hash:   bus.signingKeyBase32,
		Image:  user.Image,
		Active: true,
		Admin:  true,
		Synced: time.Now().Unix(),
	}
	return bus.droneDB.Where(droneUser).FirstOrCreate(droneUser).Error
}

func (bus BanzaiUserStorer) updateUserInDroneDB(user *User, githubAccessToken string) error {
	where := &DroneUser{
		Login: user.Login,
	}
	update := &DroneUser{
		Token:  githubAccessToken,
		Synced: time.Now().Unix(),
	}
	return bus.droneDB.Model(&DroneUser{}).Where(where).Update(update).Error
}

// This method tries to call the Drone API on a best effort basis to fetch all repos before the user navigates there.
func synchronizeDroneRepos(login string) {
	droneClient, err := NewTemporaryDroneClient(login)
	if err != nil {
		log.Warnln("failed to create Drone client", err.Error())
	}
	_, err = droneClient.RepoListOpts(true, true)
	if err != nil {
		log.Warnln("failed to sync Drone repositories", err.Error())
	}
}

func importGithubOrganizations(db *gorm.DB, currentUser *User, githubToken string) (map[uint]bool, error) {
	orgs, err := getGithubOrganizations(githubToken)
	if err != nil {
		return nil, err
	}

	orgIDs := make(map[uint]bool, len(orgs))

	tx := db.Begin()
	for _, org := range orgs {
		o := Organization{
			Name:     org.name,
			GithubID: &org.id,
			Role:     org.role,
		}

		err := tx.Where(o).First(&o).Error
		if err == nil {
			orgIDs[o.ID] = false

			continue
		} else if !gorm.IsRecordNotFoundError(err) {
			tx.Rollback()

			return nil, errors.Wrap(err, "failed to check if organization exists")
		}

		err = tx.Where(o).Create(&o).Error
		if err != nil {
			tx.Rollback()

			return nil, errors.Wrap(err, "failed to create organization")
		}

		orgIDs[o.ID] = true

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

	err = tx.Commit().Error
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

// GetUserById returns user
func GetUserById(userId uint) (*User, error) {
	db := config.DB()
	var user User
	err := db.Find(&user, User{ID: userId}).Error
	return &user, err
}

// GetUserNickNameById returns user's login name
func GetUserNickNameById(userId uint) (userName string) {

	log.Infof("Get username by id[%d]", userId)
	if user, err := GetUserById(userId); err != nil {
		log.Warnf("Error during getting user name: %s", err.Error())
	} else {
		userName = user.Login
	}

	return
}

func parseDroneTokenFromRequest(r *http.Request) (string, error) {
	var token = r.Header.Get("Authorization")

	// first we attempt to get the token from the
	// authorization header.
	if len(token) != 0 {
		token = r.Header.Get("Authorization")
		fmt.Sscanf(token, "Bearer %s", &token)
		return token, nil
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
