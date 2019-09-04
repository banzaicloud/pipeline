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

package api

import (
	"fmt"
	"net/http"
	"strconv"

	"emperror.dev/emperror"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/pkg/common"
)

// UserAPI implements user functions.
type UserAPI struct {
	db           *gorm.DB
	log          logrus.FieldLogger
	errorHandler emperror.Handler
}

// NewUserAPI returns a new UserAPI instance.
func NewUserAPI(db *gorm.DB, log logrus.FieldLogger, errorHandler emperror.Handler) *UserAPI {
	return &UserAPI{
		db:           db,
		log:          log,
		errorHandler: errorHandler,
	}
}

// GetCurrentUser responds with the authenticated user
func (a *UserAPI) GetCurrentUser(c *gin.Context) {
	user := auth.GetCurrentUser(c.Request)
	if user == nil {
		c.JSON(http.StatusUnauthorized, common.ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "failed to get current user",
		})
		return
	}

	err := a.db.Find(user).Error

	if err != nil {
		message := "failed to fetch user"
		a.errorHandler.Handle(emperror.Wrap(err, message))
		c.AbortWithStatusJSON(http.StatusInternalServerError, common.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: message,
			Error:   message,
		})
		return
	}

	scmToken, provider, err := auth.GetSCMToken(user.ID)

	if err != nil {
		message := "failed to fetch user's github token"
		a.errorHandler.Handle(emperror.Wrap(err, message))
		c.AbortWithStatusJSON(http.StatusInternalServerError, common.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: message,
			Error:   message,
		})
		return
	}
	var response struct {
		*auth.User
		GitHubTokenSet bool `json:"gitHubTokenSet"`
		GitLabTokenSet bool `json:"gitLabTokenSet"`
	}

	response.User = user
	if provider == auth.GithubTokenID {
		response.GitHubTokenSet = (scmToken != "")
	} else if provider == auth.GitlabTokenID {
		response.GitLabTokenSet = (scmToken != "")
	}

	c.JSON(http.StatusOK, response)
}

// GetUsers gets a user or lists all users from an organization depending on the presence of the id parameter.
func (a *UserAPI) GetUsers(c *gin.Context) {

	log.Info("Fetching users")

	organization := auth.GetCurrentOrganization(c.Request)

	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if idParam != "" && err != nil {
		message := "error parsing user id"
		a.errorHandler.Handle(emperror.Wrap(err, message))
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: message,
			Error:   message,
		})
		return
	}

	var users []auth.User

	err = a.db.Model(organization).Where(&auth.User{ID: uint(id)}).Related(&users, "Users").Error
	if err != nil {
		message := "failed to fetch users"
		a.errorHandler.Handle(emperror.Wrap(err, message))
		c.AbortWithStatusJSON(http.StatusInternalServerError, common.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: message,
			Error:   message,
		})
		return
	} else if id == 0 {
		c.JSON(http.StatusOK, users)
	} else if len(users) == 1 {
		c.JSON(http.StatusOK, users[0])
	} else if len(users) > 1 {
		message := fmt.Sprintf("multiple users found with id: %d", id)
		log.Info(message)
		c.AbortWithStatusJSON(http.StatusConflict, common.ErrorResponse{
			Code:    http.StatusConflict,
			Message: message,
			Error:   message,
		})
	} else {
		message := fmt.Sprintf("user not found with id: %d", id)
		log.Info(message)
		c.AbortWithStatusJSON(http.StatusNotFound, common.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: message,
			Error:   message,
		})
	}
}

type updateUserRequest struct {
	GitHubToken *string `json:"gitHubToken,omitempty"`
	GitLabToken *string `json:"gitLabToken,omitempty"`
}

// UpdateCurrentUser updates the authenticated user's settings
func (a *UserAPI) UpdateCurrentUser(c *gin.Context) {
	user := auth.GetCurrentUser(c.Request)
	if user == nil {
		c.JSON(http.StatusUnauthorized, common.ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "failed to get current user",
		})
		return
	}

	var updateUserRequest updateUserRequest
	err := c.BindJSON(&updateUserRequest)

	if err != nil {
		message := "failed to bind update user request"
		c.AbortWithStatusJSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: message,
			Error:   message,
		})
		return
	}

	err = a.db.Find(user).Error

	if err != nil {
		message := "failed to fetch user"
		a.errorHandler.Handle(emperror.Wrap(err, message))
		c.AbortWithStatusJSON(http.StatusInternalServerError, common.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: message,
			Error:   message,
		})
		return
	}

	var provider string
	var scmToken string
	if updateUserRequest.GitHubToken != nil {
		provider = auth.GithubTokenID
		scmToken = *updateUserRequest.GitHubToken
	} else if updateUserRequest.GitLabToken != nil {
		provider = auth.GitlabTokenID
		scmToken = *updateUserRequest.GitLabToken
	}

	message, err := auth.UpdateSCMToken(user, scmToken, provider)
	if err != nil {
		a.errorHandler.Handle(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, common.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: message,
			Error:   message,
		})
	}

	c.Status(http.StatusNoContent)
}
