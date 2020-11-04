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
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"emperror.dev/errors"
	helper "github.com/banzaicloud/gin-utilz/auth"
	"github.com/jinzhu/copier"
	"github.com/jinzhu/gorm"
	"github.com/qor/auth"
	"github.com/qor/auth/auth_identity"
	"github.com/qor/qor/utils"

	"github.com/banzaicloud/pipeline/internal/global"
)

const (
	// CurrentOrganization current organization key
	CurrentOrganization utils.ContextKey = "org"

	currentOrganizationID utils.ContextKey = "orgID"

	// SignUp is present if the current request is a signing up
	SignUp utils.ContextKey = "signUp"

	// OAuthRefreshTokenID denotes the tokenID for the user's OAuth refresh token, there can be only one
	OAuthRefreshTokenID = "oauth_refresh"
)

// AuthIdentity auth identity session model
type AuthIdentity struct {
	ID        uint      `gorm:"primary_key" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	auth_identity.Basic
}

// User struct
type User struct {
	ID             uint           `gorm:"primary_key" json:"id"`
	CreatedAt      *time.Time     `json:"createdAt,omitempty"`
	UpdatedAt      *time.Time     `json:"updatedAt,omitempty"`
	Name           string         `form:"name" json:"name,omitempty"`
	Email          string         `form:"email" json:"email,omitempty"`
	Login          string         `gorm:"unique;not null" form:"login" json:"login"`
	Image          string         `form:"image" json:"image,omitempty"`
	Organizations  []Organization `gorm:"many2many:user_organizations" json:"organizations,omitempty"`
	Virtual        bool           `json:"-" gorm:"-"` // Used only internally
	APIToken       string         `json:"-" gorm:"-"` // Used only internally
	ServiceAccount bool           `json:"-" gorm:"-"` // Used only internally
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
			apiToken, _ := helper.Oauth2TokenExtractor{}.ExtractToken(req)
			currentUser.APIToken = apiToken
		}
		return currentUser
	}
	return nil
}

// GetCurrentUserID returns the current user ID.
func GetCurrentUserID(req *http.Request) uint {
	user := GetCurrentUser(req)
	if user != nil {
		return user.ID
	}

	return 0
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

// BanzaiUserStorer struct
type BanzaiUserStorer struct {
	auth.UserStorer

	db        *gorm.DB
	orgSyncer OIDCOrganizationSyncer
}

// Save differs from the default UserStorer.Save() in that it
// extracts Token and Login
func (bus BanzaiUserStorer) Save(schema *auth.Schema, authCtx *auth.Context) (user interface{}, userID string, err error) {
	currentUser := &User{}
	err = copier.Copy(currentUser, schema)
	if err != nil {
		return nil, "", err
	}

	// According to the OIDC Core spec this might not always be unique,
	// but we will always use providers that are either known to provide unique usernames here,
	// or providers that we require to do that (eg. LDAP).
	currentUser.Login = schema.RawInfo.(*IDTokenClaims).PreferredUsername
	if currentUser.Login == "" {
		// When the provider does not include the preferred_username claim in the ID token,
		// fallback to generating one from the email address.
		currentUser.Login = schema.Email
	}

	// TODO: leave this to the UI?
	currentUser.Image = checkGravatarImage(currentUser.Email)

	err = bus.db.Create(currentUser).Error
	if err != nil {
		return nil, "", errors.WrapIf(err, "failed to create user organization")
	}

	err = bus.orgSyncer.SyncOrganizations(authCtx.Request.Context(), *currentUser, schema.RawInfo.(*IDTokenClaims))

	return currentUser, fmt.Sprint(bus.db.NewScope(currentUser).PrimaryKeyValue()), err
}

func checkGravatarImage(email string) string {
	h := md5.New()
	_, _ = io.WriteString(h, strings.ToLower(email))

	imageUrl := fmt.Sprintf("https://www.gravatar.com/avatar/%x?s=200", h.Sum(nil))

	imageReq, err := http.NewRequest(http.MethodHead, imageUrl, nil)
	if err != nil {
		return ""
	}

	query := imageReq.URL.Query()
	query.Set("d", "404")

	imageReq.URL.RawQuery = query.Encode()

	resp, err := http.DefaultClient.Do(imageReq)
	if err != nil {
		return ""
	}

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	return imageUrl
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

// GetOrganizationById returns an organization from database by ID
func GetOrganizationById(orgID uint) (*Organization, error) {
	db := global.DB()
	var org Organization
	err := db.Find(&org, Organization{ID: orgID}).Error
	return &org, err
}

// GetUserById returns user
func GetUserById(userId uint) (*User, error) {
	db := global.DB()
	var user User
	err := db.Find(&user, User{ID: userId}).Error
	return &user, err
}

// GetUserNickNameById returns user's login name
func GetUserNickNameById(userId uint) (userName string) {
	if userId == 0 {
		return
	}

	if user, err := GetUserById(userId); err != nil {
		log.Warn(fmt.Sprintf("Error during getting user name: %s", err.Error()))
	} else {
		userName = user.Login
	}

	return
}
