package auth

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/banzaicloud/pipeline/model"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/jinzhu/copier"
	"github.com/jinzhu/gorm"
	// blank import is used here for sql driver inclusion
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/qor/auth"
	"github.com/qor/auth/claims"
	"github.com/qor/qor/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	CurrentOrganization utils.ContextKey = "org"
)

//User struct
type User struct {
	gorm.Model
	Name          string         `form:"name"`
	Email         string         `form:"email"`
	Login         string         `gorm:"unique;not null" form:"login"`
	Image         string         `form:"image"`
	Organizations []Organization `gorm:"many2many:user_organizations"`
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

type UserOrganization struct {
	Role string `gorm:"DEFAULT:\"admin\""`
}

//Organization struct
type Organization struct {
	ID        uint                 `gorm:"primary_key" json:"id"`
	CreatedAt time.Time            `json:"createdAt"`
	UpdatedAt time.Time            `json:"updatedAt"`
	DeletedAt *time.Time           `sql:"index" json:"deletedAt,omitempty"`
	Name      string               `gorm:"unique;not null" json:"name"`
	Users     []User               `gorm:"many2many:user_organizations" json:"users,omitempty"`
	Clusters  []model.ClusterModel `gorm:"foreignkey:organization_id" json:"clusters,omitempty"`
}

//TableName sets DroneUser's table name
func (DroneUser) TableName() string {
	return "users"
}

func GetCurrentUser(req *http.Request) *User {
	if currentUser, ok := Auth.GetCurrentUser(req).(*User); ok {
		return currentUser
	}
	return nil
}

func GetCurrentOrganization(req *http.Request) *Organization {
	if organization := req.Context().Value(CurrentOrganization); organization != nil {
		return organization.(*Organization)
	}
	return nil
}

func GetCurrentUserFromDB(req *http.Request) (*User, error) {
	if currentUser, ok := Auth.GetCurrentUser(req).(*User); ok {
		claims := &claims.Claims{UserID: strconv.Itoa(int(currentUser.ID))}
		context := &auth.Context{Auth: Auth, Claims: claims, Request: req}
		if user, err := Auth.UserStorer.Get(claims, context); err == nil {
			return user.(*User), nil
		}
	}
	return nil, nil
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
	log = logger.WithFields(logrus.Fields{"tag": "Auth"})
	var tx = context.Auth.GetDB(context.Request)

	if context.Auth.Config.UserModel != nil {
		currentUser := &User{}
		copier.Copy(currentUser, schema)

		// This assumes GitHub auth only right now
		githubExtraInfo := schema.RawInfo.(*GithubExtraInfo)
		currentUser.Login = githubExtraInfo.Login
		if viper.GetBool("drone.enabled") {
			err = bus.createUserInDroneDB(currentUser, githubExtraInfo.Token)
			if err != nil {
				log.Info(context.Request.RemoteAddr, err.Error())
				return nil, "", err
			}
			bus.synchronizeDroneRepos(currentUser.Login)
		}

		// When a user registers a default organization is created in which he/she is admin
		userOrg := Organization{
			Name: currentUser.Login + "'s Org",
		}
		currentUser.Organizations = []Organization{userOrg}

		err = tx.Create(currentUser).Error
		return currentUser, fmt.Sprint(tx.NewScope(currentUser).PrimaryKeyValue()), err
	}
	return nil, "", nil
}

//http://127.0.0.1:8000/

func (bus BanzaiUserStorer) createUserInDroneDB(user *User, githubAccessToken string) error {
	droneUser := DroneUser{
		Login:  user.Login,
		Email:  user.Email,
		Token:  githubAccessToken,
		Hash:   bus.signingKeyBase32,
		Image:  user.Image,
		Active: true,
		Synced: time.Now().Unix(),
	}
	return bus.droneDB.Create(&droneUser).Error
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
	claims := &DroneClaims{Type: DroneUserCookieType, Text: login}
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
