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

package anchore

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"time"

	"github.com/banzaicloud/pipeline/internal/platform/database"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/go-errors/errors"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres" // blank import is used here for simplicity
	"github.com/sethvargo/go-password/password"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var AnchorEndpoint string
var AnchorEnabled bool

//AnchoreError
type AnchoreError struct {
	Detail   interface{} `json:"detail"`
	HttpCode int         `json:"httpcode"`
	Message  string      `json:"message"`
}

func init() {
	AnchorEndpoint = viper.GetString("anchore.endPoint")
	AnchorEnabled = viper.GetBool("anchore.enabled")
	logger = Logger()
}

type AnchoreDB struct {
	database *gorm.DB
}

func NewAnchoreDB() *AnchoreDB {
	config := database.Config{
		Host:      viper.GetString("anchore.host"),
		Port:      viper.GetInt("anchore.port"),
		User:      viper.GetString("anchore.user"),
		Pass:      viper.GetString("anchore.password"),
		Name:      viper.GetString("anchore.dbname"),
		EnableLog: viper.GetBool("anchore.logging"),
	}

	err := config.Validate()
	if err != nil {
		logger.Panic("invalid Anchore database config: ", err.Error())
	}

	dbConnStr := fmt.Sprintf("host=%v port=%v user=%v dbname=%v password=%v sslmode=disable", config.Host, config.Port, config.User, config.Name, config.Pass)
	logrus.Info(dbConnStr)
	db, err := gorm.Open("postgres", dbConnStr)
	if err != nil {
		logger.Panic("failed to initialize db: ", err.Error())
	}
	return &AnchoreDB{database: db}
}

type User struct {
	UserId         string `gorm:"primary_key;column:userId"`
	CreatedAt      int64
	LastUpdated    int64
	RecordStateKey string
	RecordStateVal string
	Password       string
	Email          string
	Acls           string
	Active         bool
}

func (db *AnchoreDB) findAnchoreUser(name string) (*User, error) {
	user := User{UserId: name}
	err := db.database.Where(&user).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (db *AnchoreDB) createAnchoreUser(name string, password string) error {
	user := User{
		UserId:         name,
		Password:       password,
		CreatedAt:      time.Now().Unix(),
		LastUpdated:    time.Now().Unix(),
		Email:          "banzai@banzaicloud.com",
		Active:         true,
		RecordStateKey: "active",
	}
	err := db.database.Create(&user).Error
	if err != nil {
		return err
	}
	return nil
}

//DeleteAnchoreUser
func (db *AnchoreDB) DeleteAnchoreUser(name string) error {
	user := User{
		UserId: name,
	}
	err := db.database.Delete(&user).Error
	if err != nil {
		return err
	}
	return nil
}

//SetupAnchoreUser sets up a new user in Anchore Postgres DB & creates / updates a secret containng user name /password.
func SetupAnchoreUser(orgId uint, clusterId string) (bool, error) {
	if !AnchorEnabled {
		logger.Infof("Anchore integration is not enabled.")
		return false, nil
	}
	anchorUserName := fmt.Sprintf("%v-anchore-user", clusterId)
	db := NewAnchoreDB()
	defer db.database.Close()

	user, err := db.findAnchoreUser(anchorUserName)
	var userPassword string
	if err != nil {
		logger.Infof("Anchore user %v not found, creating", anchorUserName)
		userPassword, err = password.Generate(16, 4, 4, false, true)
		if err != nil {
			return true, errors.WrapPrefix(err, "Error generating password for Anchor user", 0)
		}
		err := db.createAnchoreUser(anchorUserName, userPassword)
		if err != nil {
			return true, errors.WrapPrefix(err, "Error creating Anchor user", 0)
		}
	} else {
		logger.Infof("Anchore user %v found", anchorUserName)
		userPassword = user.Password
	}

	secretRequest := secret.CreateSecretRequest{
		Name: anchorUserName,
		Type: "password",
		Values: map[string]string{
			"username": anchorUserName,
			"password": userPassword,
		},
	}
	if _, err := secret.Store.CreateOrUpdate(orgId, &secretRequest); err != nil {
		return true, errors.WrapPrefix(err, "Failed to create/update Anchore user secret", 0)
	}

	return true, nil
}

func RemoveAnchoreUser(orgId uint, clusterId string) {
	if !AnchorEnabled {
		return
	}
	anchorUserName := fmt.Sprintf("%v-anchore-user", clusterId)
	db := NewAnchoreDB()
	defer db.database.Close()

	err := db.DeleteAnchoreUser(anchorUserName)
	if err != nil {
		logger.Errorf("Error deleting Anchore user: %v", err.Error())
	} else {
		logger.Infof("Anchore user %v deleted.", anchorUserName)
	}
}

func MakePolicyRequest(orgId uint, clusterId string, method string, url string, body interface{}) (*http.Response, error) {

	if !AnchorEnabled {
		return nil, errors.New("Anchore integration is not enabled. You can enable by setting config property: anchor.enabled = true.")
	}

	anchorUserName := fmt.Sprintf("%v-anchore-user", clusterId)
	anchoreUserSecret, err := secret.Store.GetByName(orgId, anchorUserName)
	if err != nil {
		return nil, err
	}

	password := anchoreUserSecret.Values["password"]

	auth := fmt.Sprintf("%v:%v", anchorUserName, password)
	sEnc := base64.StdEncoding.EncodeToString([]byte(auth))

	var request *http.Request
	if body != nil {
		var buf io.ReadWriter
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}

		request, _ = http.NewRequest(method, AnchorEndpoint+"/"+path.Join("imagecheck/v1", url), buf)
		if err != nil {
			return nil, err
		}
	} else {
		request, _ = http.NewRequest(method, AnchorEndpoint+"/"+path.Join("imagecheck/v1", url), nil)
		if err != nil {
			return nil, err
		}
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Basic %v", sEnc))
	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		logger.Error(err)
		return response, err
	}

	return response, nil
}
