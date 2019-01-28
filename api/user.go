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

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/pkg/common"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

type userAccessManager interface {
	GrantOrganizationAccessToUser(userID string, orgID uint)
	RevokeOrganizationAccessFromUser(userID string, orgID uint)
}

// UserAPI implements user functions.
type UserAPI struct {
	accessManager userAccessManager
	db            *gorm.DB
}

// NewUserAPI returns a new UserAPI instance.
func NewUserAPI(accessManager userAccessManager, db *gorm.DB) *UserAPI {
	return &UserAPI{
		accessManager: accessManager,
		db:            db,
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
		log.Info(message + ": " + err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, common.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: message,
			Error:   message,
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

// GetUsers gets a user or lists all users from an organization depending on the presence of the id parameter.
func (a *UserAPI) GetUsers(c *gin.Context) {

	log.Info("Fetching users")

	organization := auth.GetCurrentOrganization(c.Request)

	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if idParam != "" && err != nil {
		message := fmt.Sprintf("error parsing user id: %s", err)
		log.Info(message)
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
		log.Info(message + ": " + err.Error())
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

// AddUser adds a user to an organization, role=admin|member has to be in the body, otherwise member is the default role.
func (a *UserAPI) AddUser(c *gin.Context) {

	log.Info("Adding user to organization")

	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		message := fmt.Sprintf("error parsing user id: %s", err)
		log.Info(message)
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: message,
			Error:   message,
		})
		return
	}

	role := struct {
		Role string `json:"role" binding:"required,eq=member|eq=admin"`
	}{Role: "member"}

	if c.Request.ContentLength != 0 {
		err = c.ShouldBindJSON(&role)
		if err != nil {
			message := fmt.Sprintf("error parsing role from request: %s", err)
			log.Info(message)
			c.JSON(http.StatusBadRequest, common.ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: message,
				Error:   message,
			})
			return
		}
	}

	organization := auth.GetCurrentOrganization(c.Request)
	user := &auth.User{ID: uint(id)}

	err = a.addUserToOrgInDb(organization, user, role.Role)

	if err != nil {
		message := "failed to add user: " + err.Error()
		log.Info(message)
		statusCode := auth.GormErrorToStatusCode(err)
		c.AbortWithStatusJSON(statusCode, common.ErrorResponse{
			Code:    statusCode,
			Message: message,
			Error:   message,
		})
		return
	}

	a.accessManager.GrantOrganizationAccessToUser(user.IDString(), organization.ID)

	c.Status(http.StatusNoContent)
}

func (a *UserAPI) addUserToOrgInDb(organization *auth.Organization, user *auth.User, role string) error {
	tx := a.db.Begin()
	err := tx.Error
	if err != nil {
		tx.Rollback()
		return err
	}
	err = tx.Model(organization).Association("Users").Append(user).Error
	if err != nil {
		tx.Rollback()
		return err
	}
	userRoleInOrg := auth.UserOrganization{UserID: user.ID, OrganizationID: organization.ID}
	err = tx.Model(&auth.UserOrganization{}).Where(userRoleInOrg).Update("role", role).Error
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

// RemoveUser removes a user from an organization
func (a *UserAPI) RemoveUser(c *gin.Context) {

	log.Info("Deleting user from organization")

	organization := auth.GetCurrentOrganization(c.Request)

	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		message := fmt.Sprintf("error parsing user id: %s", err)
		log.Info(message)
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: message,
			Error:   message,
		})
		return
	}

	err = a.db.Model(organization).Association("Users").Delete(auth.User{ID: uint(id)}).Error
	if err != nil {
		message := "failed to delete user: " + err.Error()
		log.Info(message)
		statusCode := auth.GormErrorToStatusCode(err)
		c.AbortWithStatusJSON(statusCode, common.ErrorResponse{
			Code:    statusCode,
			Message: message,
			Error:   message,
		})
		return
	}

	user := auth.GetCurrentUser(c.Request)

	a.accessManager.RevokeOrganizationAccessFromUser(user.IDString(), organization.ID)

	c.Status(http.StatusNoContent)
}

type updateUserRequest struct {
	GitHubToken string `json:"gitHubToken,omitempty"`
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
		log.Info(message + ": " + err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, common.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: message,
			Error:   message,
		})
		return
	}

	if updateUserRequest.GitHubToken != "" {
		err = auth.SaveUserGitHubToken(user, updateUserRequest.GitHubToken)
		if err != nil {
			message := "failed to update user's github token"
			log.Info(message + ": " + err.Error())
			c.AbortWithStatusJSON(http.StatusInternalServerError, common.ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: message,
				Error:   message,
			})
			return
		}
	}

	c.Status(http.StatusNoContent)
}
