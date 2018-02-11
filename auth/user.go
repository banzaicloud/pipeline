package auth

import (
	"fmt"
	"net/http"
	"time"

	banzaiConstants "github.com/banzaicloud/banzai-types/constants"
	banzaiUtils "github.com/banzaicloud/banzai-types/utils"
	"github.com/jinzhu/copier"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/qor/auth"
	"github.com/spf13/viper"
)

type User struct {
	gorm.Model
	Name  string `form:"name"`
	Email string `form:"email"`
	Login string `form:"login"`
	Image string `form:"image"`
}

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

func (DroneUser) TableName() string {
	return "users"
}

func getCurrentUser(req *http.Request) *User {
	if currentUser, ok := Auth.GetCurrentUser(req).(*User); ok {
		return currentUser
	}
	return nil
}

type BanzaiUserStorer struct {
	auth.UserStorer
	signingKeyBase32 string // Drone uses base32 Hash
	droneDB          *gorm.DB
}

// This differs from the default UserStorer.Save() in that it
// extracts Token and Login and saves to Drone DB as well
func (bus BanzaiUserStorer) Save(schema *auth.Schema, context *auth.Context) (user interface{}, userID string, err error) {
	var tx = context.Auth.GetDB(context.Request)

	if context.Auth.Config.UserModel != nil {
		currentUser := &User{}
		copier.Copy(currentUser, schema)
		// This assumes GitHub auth only right now
		githubExtraInfo := schema.RawInfo.(*GithubExtraInfo)
		currentUser.Login = githubExtraInfo.Login
		err = bus.createUserInDroneDB(currentUser, githubExtraInfo.Token)
		if err != nil {
			banzaiUtils.LogInfo(banzaiConstants.TagAuth, context.Request.RemoteAddr, err.Error())
			return nil, "", err
		}
		err = tx.Create(currentUser).Error
		return currentUser, fmt.Sprint(tx.NewScope(currentUser).PrimaryKeyValue()), err
	}
	return nil, "", nil
}

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

func initDroneDatabase() *gorm.DB {
	host := viper.GetString("dev.host")
	port := viper.GetString("dev.port")
	user := viper.GetString("dev.user")
	password := viper.GetString("dev.password")

	db, err := gorm.Open("mysql", user+":"+password+"@tcp("+host+":"+port+")/drone?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		panic(err)
	}
	db.LogMode(true)

	return db
}
