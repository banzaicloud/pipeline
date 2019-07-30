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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path"

	"emperror.dev/emperror"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	secretTypes "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
)

var AnchoreEndpoint string  // nolint: gochecknoglobals
var AnchoreEnabled bool     // nolint: gochecknoglobals
var AnchoreAdminUser string // nolint: gochecknoglobals
var AnchoreAdminPass string // nolint: gochecknoglobals

const (
	anchoreEmail                  string = "banzai@banzaicloud.com"
	accountPath                   string = "accounts"
	SecurityScanNotEnabledMessage string = "security scan isn't enabled"
)

// AnchoreError
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
		URL:       accountPath,
		Body:      anchoreAccount,
	}
	_, err := DoAnchoreRequest(anchoreRequest)
	if err != nil {
		return emperror.Wrap(err, "account create AnchoreRequest failed")
	}
	return nil
}

func createAnchoreUser(username string, password string) error {
	anchoreUser := anchoreUserPostBody{
		Username: username,
		Password: password,
	}
	endPoint := path.Join(accountPath, username, "users")
	anchoreRequest := AnchoreRequest{
		AdminUser: true,
		Method:    http.MethodPost,
		URL:       endPoint,
		Body:      anchoreUser,
	}
	_, err := DoAnchoreRequest(anchoreRequest)
	if err != nil {
		return emperror.Wrap(err, "user create AnchoreRequest failed")
	}
	return nil
}

func checkAnchoreUser(username string, method string) int {
	endPoint := anchoreUserEndPoint(username)
	anchoreRequest := AnchoreRequest{
		AdminUser: true,
		Method:    method,
		URL:       endPoint,
	}
	response, err := DoAnchoreRequest(anchoreRequest)
	if err != nil {
		logger.Error(err)
		return http.StatusInternalServerError
	}
	return response.StatusCode
}

func deleteAnchoreAccount(account string) int {
	endPoint := path.Join(accountPath, account)
	type accountStatus struct {
		State string `json:"state"`
	}

	accStatus := accountStatus{
		State: "disabled",
	}
	anchoreRequest := AnchoreRequest{
		AdminUser: true,
		Method:    http.MethodPut,
		URL:       path.Join(endPoint, "state"),
		Body:      accStatus,
	}
	response, err := DoAnchoreRequest(anchoreRequest)
	if err != nil {
		logger.Error(err)
		return http.StatusInternalServerError
	}

	anchoreRequest = AnchoreRequest{
		AdminUser: true,
		Method:    http.MethodDelete,
		URL:       endPoint,
		Body:      nil,
	}
	response, err = DoAnchoreRequest(anchoreRequest)
	if err != nil {
		logger.Error(err)
		return http.StatusInternalServerError
	}
	return response.StatusCode
}

func getAnchoreUserCredentials(username string) (string, int) {
	type userCred struct {
		Type      string `json:"type"`
		Value     string `json:"value"`
		CreatedAt string `json:"created_at"`
	}

	endPoint := path.Join(anchoreUserEndPoint(username), "credentials")
	anchoreRequest := AnchoreRequest{
		AdminUser: true,
		Method:    http.MethodGet,
		URL:       endPoint,
	}
	response, err := DoAnchoreRequest(anchoreRequest)
	if err != nil {
		logger.Error(err)
		return "", http.StatusInternalServerError
	}
	defer response.Body.Close()
	var usercreds userCred
	respBody, _ := ioutil.ReadAll(response.Body)
	json.Unmarshal(respBody, &usercreds) // nolint: errcheck
	userPass := usercreds.Value

	return userPass, response.StatusCode
}

func anchoreUserEndPoint(username string) string {
	return path.Join(accountPath, username, "users", username)
}

// SetupAnchoreUser sets up a new user in Anchore Postgres DB & creates / updates a secret containng user name /password.
func SetupAnchoreUser(orgId uint, clusterId string) (*User, error) {
	anchoreUserName := fmt.Sprintf("%v-anchore-user", clusterId)
	var user User
	if checkAnchoreUser(anchoreUserName, http.MethodGet) != http.StatusOK {
		logger.Infof("Anchore user %v not found, creating", anchoreUserName)

		secretRequest := secret.CreateSecretRequest{
			Name: anchoreUserName,
			Type: "password",
			Values: map[string]string{
				"username": anchoreUserName,
				"password": "",
			},
			Tags: []string{
				secretTypes.TagBanzaiHidden,
			},
		}
		secretId, err := secret.Store.CreateOrUpdate(orgId, &secretRequest)
		if err != nil {
			return nil, emperror.Wrap(err, "failed to create/update Anchore user secret")
		}
		// retrieve crated secret to read generated password
		secretItem, err := secret.Store.Get(orgId, secretId)
		if err != nil {
			return nil, emperror.Wrap(err, "failed to retrieve Anchore user secret")
		}
		userPassword := secretItem.Values["password"]

		if createAnchoreAccount(anchoreUserName, anchoreEmail) != nil {
			return nil, emperror.Wrap(err, "error creating Anchor account")
		}
		if createAnchoreUser(anchoreUserName, userPassword) != nil {
			return nil, emperror.Wrap(err, "error creating Anchor user")
		}
		user.Password = userPassword
		user.UserId = anchoreUserName
	} else {
		logger.Infof("Anchore user %v found", anchoreUserName)
		userPassword, status := getAnchoreUserCredentials(anchoreUserName)
		if status != http.StatusOK {
			var err error
			return nil, emperror.Wrap(err, "failed to get Anchore user secret")
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
			return nil, emperror.Wrap(err, "failed to create/update Anchore user secret")
		}
	}

	return &user, nil
}

func RemoveAnchoreUser(orgId uint, clusterId string) {

	anchorUserName := fmt.Sprintf("%v-anchore-user", clusterId)

	secretItem, err := secret.Store.GetByName(orgId, anchorUserName)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"organization": orgId,
			"user":         anchorUserName,
		}).Info("error fetching Anchore user secret")
		return
	}
	err = secret.Store.Delete(orgId, secretItem.ID)
	if err != nil {
		logger.Error(emperror.WrapWith(err, "error deleting Anchore user secret", "organization", orgId, "user", anchorUserName))
	} else {
		logger.WithFields(logrus.Fields{
			"organization": orgId,
			"user":         anchorUserName,
		}).Debug("Anchore user secret deleted")
	}
	if checkAnchoreUser(anchorUserName, http.MethodDelete) != http.StatusNoContent {
		logger.Error(emperror.WrapWith(err, "error deleting Anchore user", "organization", orgId, "user", anchorUserName))
	}
	logger.WithFields(logrus.Fields{
		"organization": orgId,
		"user":         anchorUserName,
	}).Debug("Anchore user secret deleted")
	if deleteAnchoreAccount(anchorUserName) != http.StatusNoContent {
		logger.Error(emperror.WrapWith(err, "error deleting Anchore account", "organization", orgId, "account", anchorUserName))
	}
	logger.WithFields(logrus.Fields{
		"organization": orgId,
		"account":      anchorUserName,
	}).Debug("Anchore account deleted")
}

// DoAnchoreRequest do anchore api call
func DoAnchoreRequest(req AnchoreRequest) (*http.Response, error) {

	if !AnchoreEnabled {
		return nil, errors.New("anchore integration is not enabled, you can enable by setting config property: anchor.enabled = true")
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
			return nil, emperror.Wrap(err, "failed to get secret")
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
			return nil, emperror.Wrap(err, "json encode failed")
		}
		request, err = http.NewRequest(req.Method, AnchoreEndpoint+path.Join("/v1", req.URL), buf)
		if err != nil {
			return nil, emperror.Wrap(err, "request creation failed")
		}
	} else {
		var err error
		request, err = http.NewRequest(req.Method, AnchoreEndpoint+path.Join("/v1", req.URL), nil)
		if err != nil {
			return nil, emperror.Wrap(err, "request creation failed")
		}
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Basic %v", sEnc))
	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		return nil, emperror.Wrap(err, "anchore request failed")
	}

	return response, nil
}
