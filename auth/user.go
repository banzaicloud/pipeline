package auth

import (
	"fmt"
	"net/http"
	"time"

	"github.com/banzaicloud/pipeline/model"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-errors/errors"
	"github.com/google/go-github/github"
	"github.com/jinzhu/copier"
	"github.com/jinzhu/gorm"
	"golang.org/x/oauth2"
	// blank import is used here for sql driver inclusion
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/qor/auth"
	"github.com/qor/auth/claims"
	"github.com/qor/qor/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	// CurrentOrganization current organization key
	CurrentOrganization utils.ContextKey = "org"
)

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
	ID        uint                 `gorm:"primary_key" json:"id"`
	GithubID  *int64               `gorm:"unique" json:"githubId,omitempty"`
	CreatedAt time.Time            `json:"createdAt"`
	UpdatedAt time.Time            `json:"updatedAt"`
	Name      string               `gorm:"unique,not null" json:"name"`
	Users     []User               `gorm:"many2many:user_organizations" json:"users,omitempty"`
	Clusters  []model.ClusterModel `gorm:"foreignkey:organization_id" json:"clusters,omitempty"`
	Role      string               `json:"-" gorm:"-"` // Used only internally
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

// GetCurrentUserFromDB returns the current user from the database
func GetCurrentUserFromDB(req *http.Request) (*User, error) {
	if currentUser, ok := Auth.GetCurrentUser(req).(*User); ok {
		claims := &claims.Claims{UserID: currentUser.IDString()}
		context := &auth.Context{Auth: Auth, Claims: claims, Request: req}
		user, err := Auth.UserStorer.Get(claims, context)
		if err != nil {
			return nil, err
		}
		return user.(*User), nil
	}
	return nil, errors.New("error fetching user from db")
}

//BanzaiUserStorer struct
type BanzaiUserStorer struct {
	auth.UserStorer
	signingKeyBase32 string // Drone uses base32 Hash
	droneDB          *gorm.DB
}

// Save differs from the default UserStorer.Save() in that it
// extracts Token and Login and saves to Drone DB as well
func (bus BanzaiUserStorer) Save(schema *auth.Schema, context *auth.Context) (user interface{}, userID string, err error) {
	log := logger.WithFields(logrus.Fields{"tag": "Auth"})

	currentUser := &User{}
	copier.Copy(currentUser, schema)

	// This assumes GitHub auth only right now
	githubExtraInfo := schema.RawInfo.(*GithubExtraInfo)
	currentUser.Login = githubExtraInfo.Login
	err = bus.createUserInDroneDB(currentUser, githubExtraInfo.Token)
	if err != nil {
		log.Info(context.Request.RemoteAddr, err.Error())
		return nil, "", err
	}
	bus.synchronizeDroneRepos(currentUser.Login)

	// When a user registers a default organization is created in which he/she is admin
	userOrg := Organization{
		Name: currentUser.Login,
	}
	currentUser.Organizations = []Organization{userOrg}

	db := context.Auth.GetDB(context.Request)
	err = db.Create(currentUser).Error
	if err != nil {
		return nil, "", err
	}

	AddDefaultRoleForUser(currentUser.ID)

	githubOrgIDs, err := importGithubOrganizations(currentUser, context, githubExtraInfo.Token)

	if err == nil {
		orgids := []uint{currentUser.Organizations[0].ID}
		orgids = append(orgids, githubOrgIDs...)
		AddOrgRoles(orgids...)
		AddOrgRoleForUser(currentUser.ID, orgids...)
	}

	return currentUser, fmt.Sprint(db.NewScope(currentUser).PrimaryKeyValue()), err
}

func (bus BanzaiUserStorer) createUserInDroneDB(user *User, githubAccessToken string) error {
	droneUser := &DroneUser{
		Login:  user.Login,
		Email:  user.Email,
		Token:  githubAccessToken,
		Hash:   bus.signingKeyBase32,
		Image:  user.Image,
		Active: true,
		Synced: time.Now().Unix(),
	}
	return bus.droneDB.Where(droneUser).FirstOrCreate(droneUser).Error
}

func initDroneDB() *gorm.DB {
	return model.ConnectDB("drone")
}

// This method tries to call the Drone API on a best effort basis to fetch all repos before the user navigates there.
func (bus BanzaiUserStorer) synchronizeDroneRepos(login string) {
	droneURL := viper.GetString("drone.url")
	req, err := http.NewRequest("GET", droneURL+"/api/user/repos?all=true&flush=true", nil)
	if err != nil {
		log.Info("synchronizeDroneRepos: failed to create Drone GET request", err.Error())
		return
	}

	// Create a temporary Drone API token
	claims := &DroneClaims{Type: DroneUserTokenType, Text: login}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	apiToken, err := token.SignedString([]byte(bus.signingKeyBase32))
	if err != nil {
		log.Info("synchronizeDroneRepos: failed to create temporary token for Drone GET request", err.Error())
		return
	}
	req.Header.Add("Authorization", "Bearer "+apiToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Info("synchronizeDroneRepos: failed to call Drone API", err.Error())
		return
	}
	if resp.StatusCode != http.StatusOK {
		log.Info("synchronizeDroneRepos: failed to call Drone API HTTP", resp.StatusCode)
	}
}

func getGithubOrganizations(token string) ([]*Organization, error) {
	httpClient := oauth2.NewClient(
		oauth2.NoContext,
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}),
	)
	githubClient := github.NewClient(httpClient)

	memberships, _, err := githubClient.Organizations.ListOrgMemberships(oauth2.NoContext, nil)
	if err != nil {
		return nil, err
	}

	orgs := []*Organization{}
	for _, membership := range memberships {
		githubOrg := membership.GetOrganization()
		org := Organization{Name: githubOrg.GetLogin(), GithubID: githubOrg.ID, Role: membership.GetRole()}
		orgs = append(orgs, &org)
	}
	return orgs, nil
}

func importGithubOrganizations(currentUser *User, context *auth.Context, githubToken string) ([]uint, error) {

	githubOrgs, err := getGithubOrganizations(githubToken)
	if err != nil {
		log.Info("Failed to list organizations", err)
		githubOrgs = []*Organization{}
	}

	orgids := []uint{}

	tx := context.Auth.GetDB(context.Request).Begin()
	{
		for _, githubOrg := range githubOrgs {
			err = tx.Where(&githubOrg).FirstOrCreate(githubOrg).Error
			if err != nil {
				tx.Rollback()
				return nil, err
			}
			err = tx.Model(currentUser).Association("Organizations").Append(githubOrg).Error
			if err != nil {
				tx.Rollback()
				return nil, err
			}
			userRoleInOrg := UserOrganization{UserID: currentUser.ID, OrganizationID: githubOrg.ID}
			err = tx.Model(&UserOrganization{}).Where(userRoleInOrg).Update("role", githubOrg.Role).Error
			if err != nil {
				tx.Rollback()
				return nil, err
			}
			orgids = append(orgids, githubOrg.ID)
		}
	}

	err = tx.Commit().Error
	if err != nil {
		return nil, err
	}

	return orgids, nil
}
