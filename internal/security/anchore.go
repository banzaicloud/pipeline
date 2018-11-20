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
	"io/ioutil"
	"net/http"
	"path"

	"github.com/banzaicloud/pipeline/secret"
	"github.com/cloudflare/cfssl/log"
	"github.com/go-errors/errors"
	_ "github.com/jinzhu/gorm/dialects/postgres" // blank import is used here for simplicity
	"github.com/spf13/viper"
)

var AnchoreEndpoint string
var AnchoreEnabled bool
var AnchoreAdminUser string
var AnchoreAdminPass string

const anchoreEmail string = "banzai@banzaicloud.com"

//AnchoreError
type AnchoreError struct {
	Detail   interface{} `json:"detail"`
	HttpCode int         `json:"httpcode"`
	Message  string      `json:"message"`
}

func init() {
	AnchoreEndpoint = viper.GetString("anchore.endPoint")
	AnchoreEnabled = viper.GetBool("anchore.enabled")
	AnchoreAdminUser = viper.GetString("anchore.adminUser")
	AnchoreAdminPass = viper.GetString("anchore.adminPass")
	logger = Logger()
}

type User struct {
	UserId   string
	Password string
}

type anchoreAccountPostBody struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

type anchoreUserPostBody struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// AnchoreRequest anchore API request
type AnchoreRequest struct {
	OrgID     uint
	ClusterID string
	Method    string
	URL       string
	Body      interface{}
	AdminUser bool
}

func createAnchoreAccount(name string, email string) error {
	anchoreAccount := anchoreAccountPostBody{
		Name:  name,
		Email: email,
	}

	anchoreRequest := AnchoreRequest{
		AdminUser: true,
		Method:    http.MethodPost,
		URL:       "accounts",
		Body:      anchoreAccount,
	}
	response, err := MakeAnchoreRequest(anchoreRequest)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return err
}

func createAnchoreUser(username string, password string) error {
	anchoreUser := anchoreUserPostBody{
		Username: username,
		Password: password,
	}

	endPoint := "accounts"
	endPoint = path.Join(endPoint, username)
	endPoint = path.Join(endPoint, "users")

	anchoreRequest := AnchoreRequest{
		AdminUser: true,
		Method:    http.MethodPost,
		URL:       endPoint,
		Body:      anchoreUser,
	}
	response, err := MakeAnchoreRequest(anchoreRequest)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return err
}

func getAnchoreUser(username string) int {
	endPoint := "accounts"
	endPoint = path.Join(endPoint, username)
	endPoint = path.Join(endPoint, "users")
	endPoint = path.Join(endPoint, username)

	anchoreRequest := AnchoreRequest{
		AdminUser: true,
		Method:    http.MethodGet,
		URL:       endPoint,
	}
	response, err := MakeAnchoreRequest(anchoreRequest)
	if err != nil {
		log.Error(err)
		return response.StatusCode
	}
	defer response.Body.Close()
	return response.StatusCode
}

func deleteAnchoreUser(username string) int {
	endPoint := "accounts"
	endPoint = path.Join(endPoint, username)
	endPoint = path.Join(endPoint, "users")
	endPoint = path.Join(endPoint, username)

	anchoreRequest := AnchoreRequest{
		AdminUser: true,
		Method:    http.MethodDelete,
		URL:       endPoint,
	}
	response, err := MakeAnchoreRequest(anchoreRequest)
	if err != nil {
		log.Error(err)
		return response.StatusCode
	}
	defer response.Body.Close()
	return response.StatusCode
}

func deleteAnchoreAccount(account string) int {
	endPoint := "accounts"
	endPoint = path.Join(endPoint, account)

	anchoreRequest := AnchoreRequest{
		AdminUser: true,
		Method:    http.MethodDelete,
		URL:       endPoint,
	}
	response, err := MakeAnchoreRequest(anchoreRequest)
	if err != nil {
		log.Error(err)
		return response.StatusCode
	}
	defer response.Body.Close()
	return response.StatusCode
}

func getAnchoreUserCredentials(username string) (string, int) {
	type userCred struct {
		Type      string `json:"type"`
		Value     string `json:"value"`
		CreatedAt string `json:"created_at"`
	}

	endPoint := "accounts"
	endPoint = path.Join(endPoint, username)
	endPoint = path.Join(endPoint, "users")
	endPoint = path.Join(endPoint, username)
	endPoint = path.Join(endPoint, "credentials")

	anchoreRequest := AnchoreRequest{
		AdminUser: true,
		Method:    http.MethodGet,
		URL:       endPoint,
	}
	response, err := MakeAnchoreRequest(anchoreRequest)
	if err != nil {
		log.Error(err)
		return "", response.StatusCode
	}
	defer response.Body.Close()
	var usercreds userCred
	respBody, _ := ioutil.ReadAll(response.Body)
	json.Unmarshal(respBody, &usercreds)
	userPass := usercreds.Value

	return userPass, response.StatusCode
}

//SetupAnchoreUser sets up a new user in Anchore Postgres DB & creates / updates a secret containng user name /password.
func SetupAnchoreUser(orgId uint, clusterId string) (*User, error) {
	anchoreUserName := fmt.Sprintf("%v-anchore-user", clusterId)
	var user *User
	if getAnchoreUser(anchoreUserName) != http.StatusOK {
		logger.Infof("Anchore user %v not found, creating", anchoreUserName)

		secretRequest := secret.CreateSecretRequest{
			Name: anchoreUserName,
			Type: "password",
			Values: map[string]string{
				"username": anchoreUserName,
				"password": "",
			},
		}
		secretId, err := secret.Store.CreateOrUpdate(orgId, &secretRequest)
		if err != nil {
			return nil, errors.WrapPrefix(err, "Failed to create/update Anchore user secret", 0)
		}
		// retrieve crated secret to read generated password
		secretItem, err := secret.Store.Get(orgId, secretId)
		if err != nil {
			return nil, errors.WrapPrefix(err, "Failed to retrieve Anchore user secret", 0)
		}
		userPassword := secretItem.Values["password"]

		if createAnchoreAccount(anchoreUserName, anchoreEmail) != nil {
			return nil, errors.WrapPrefix(err, "Error creating Anchor user", 0)
		}
		if createAnchoreUser(anchoreUserName, userPassword) != nil {
			return nil, errors.WrapPrefix(err, "Error creating Anchor user", 0)
		}
		user.Password = anchoreUserName
		user.UserId = userPassword
	} else {
		logger.Infof("Anchore user %v found", anchoreUserName)
		userPassword, status := getAnchoreUserCredentials(anchoreUserName)
		if status != http.StatusOK {
			var err error
			return nil, errors.WrapPrefix(err, "Failed to get Anchore user secret", 0)
		}
		secretRequest := secret.CreateSecretRequest{
			Name: anchoreUserName,
			Type: "password",
			Values: map[string]string{
				"username": anchoreUserName,
				"password": userPassword,
			},
		}
		if _, err := secret.Store.CreateOrUpdate(orgId, &secretRequest); err != nil {
			return nil, errors.WrapPrefix(err, "Failed to create/update Anchore user secret", 0)
		}
	}

	return user, nil
}

func RemoveAnchoreUser(orgId uint, clusterId string) {
	if !AnchoreEnabled {
		return
	}
	anchorUserName := fmt.Sprintf("%v-anchore-user", clusterId)

	// err := db.DeleteAnchoreUser(anchorUserName)

	if deleteAnchoreUser(anchorUserName) != http.StatusNoContent {
		logger.Errorf("Error deleting Anchore user: %v", anchorUserName)
		return
	}
	if deleteAnchoreAccount(anchorUserName) != http.StatusNoContent {
		logger.Errorf("Error deleting Anchore account: %v", anchorUserName)
		return
	}
	logger.Infof("Anchore user %v deleted.", anchorUserName)

	secretItem, err := secret.Store.GetByName(orgId, anchorUserName)
	if err != nil {
		logger.Errorf("Error fetching Anchore user secret: %v", err.Error())
		return
	}
	err = secret.Store.Delete(orgId, secretItem.ID)
	if err != nil {
		logger.Errorf("Error deleting Anchore user secret: %v", err.Error())
	} else {
		logger.Infof("Anchore user secret %v deleted.", anchorUserName)
	}
}

// MakeAnchoreRequest do anchore api call
func MakeAnchoreRequest(req AnchoreRequest) (*http.Response, error) {

	if !AnchoreEnabled {
		return nil, errors.New("Anchore integration is not enabled. You can enable by setting config property: anchor.enabled = true.")
	}
	var anchoreUserName string
	var password string
	if req.AdminUser {
		anchoreUserName = AnchoreAdminUser
		password = AnchoreAdminPass
	} else {
		anchoreUserName = fmt.Sprintf("%v-anchore-user", req.ClusterID)
		anchoreUserSecret, err := secret.Store.GetByName(req.OrgID, anchoreUserName)
		if err != nil {
			return nil, err
		}
		password = anchoreUserSecret.Values["password"]
	}

	auth := fmt.Sprintf("%v:%v", anchoreUserName, password)
	sEnc := base64.StdEncoding.EncodeToString([]byte(auth))

	var request *http.Request
	if req.Body != nil {
		var buf io.ReadWriter
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(req.Body)
		if err != nil {
			return nil, err
		}
		request, err = http.NewRequest(req.Method, AnchoreEndpoint+"/"+path.Join("v1", req.URL), buf)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		request, err = http.NewRequest(req.Method, AnchoreEndpoint+"/"+path.Join("v1", req.URL), nil)
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
