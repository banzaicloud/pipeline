package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/model"
	"github.com/gin-gonic/gin"
)

// GetUsers gets a user or lists all users from an organizaion depending on the presence of the id paramater
func GetUsers(c *gin.Context) {

	log.Info("Fetching users")

	organization := auth.GetCurrentOrganization(c.Request)

	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if idParam != "" && err != nil {
		message := fmt.Sprintf("error parsing user id: %s", err)
		log.Info(message)
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: message,
			Error:   message,
		})
		return
	}

	var users []auth.User
	db := model.GetDB()
	err = db.Model(organization).Related(&users, "Users").Error
	if err != nil {
		message := "failed to fetch users"
		log.Info(message + ": " + err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: message,
			Error:   message,
		})
	} else if id == 0 {
		c.JSON(http.StatusOK, users)
	} else if len(users) == 1 {
		c.JSON(http.StatusOK, users[0])
	} else if len(users) > 1 {
		message := fmt.Sprintf("multiple users found with id: %d", id)
		log.Info(message)
		c.AbortWithStatusJSON(http.StatusConflict, components.ErrorResponse{
			Code:    http.StatusConflict,
			Message: message,
			Error:   message,
		})
	} else {
		message := fmt.Sprintf("user not found with id: %d", id)
		log.Info(message)
		c.AbortWithStatusJSON(http.StatusNotFound, components.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: message,
			Error:   message,
		})
	}
}

// AddUser adds a user to an organization, role=admin|member has to be in the body, otherwise member is the default role.
func AddUser(c *gin.Context) {

	log.Info("Adding user to organization")

	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		message := fmt.Sprintf("error parsing user id: %s", err)
		log.Info(message)
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
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
			c.JSON(http.StatusBadRequest, components.ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: message,
				Error:   message,
			})
			return
		}
	}

	organization := auth.GetCurrentOrganization(c.Request)
	user := &auth.User{ID: uint(id)}

	err = addUserToOrgInDb(organization, user, role.Role)

	if err != nil {
		message := "failed to add user: " + err.Error()
		log.Info(message)
		statusCode := auth.GormErrorToStatusCode(err)
		c.AbortWithStatusJSON(statusCode, components.ErrorResponse{
			Code:    statusCode,
			Message: message,
			Error:   message,
		})
		return
	}

	auth.AddOrgRoleForUser(user.ID, organization.ID)

	c.Status(http.StatusNoContent)
}

func addUserToOrgInDb(organization *auth.Organization, user *auth.User, role string) error {
	tx := model.GetDB().Begin()
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
func RemoveUser(c *gin.Context) {

	log.Info("Deleting user from organization")

	organization := auth.GetCurrentOrganization(c.Request)

	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		message := fmt.Sprintf("error parsing user id: %s", err)
		log.Info(message)
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: message,
			Error:   message,
		})
		return
	}

	db := model.GetDB()
	err = db.Model(organization).Association("Users").Delete(auth.User{ID: uint(id)}).Error
	if err != nil {
		message := "failed to delete user: " + err.Error()
		log.Info(message)
		statusCode := auth.GormErrorToStatusCode(err)
		c.AbortWithStatusJSON(statusCode, components.ErrorResponse{
			Code:    statusCode,
			Message: message,
			Error:   message,
		})
		return
	}

	user, err := auth.GetCurrentUserFromDB(c.Request)
	if err != nil {
		message := "failed to query user for deletion: " + err.Error()
		log.Info(message)
		statusCode := auth.GormErrorToStatusCode(err)
		c.AbortWithStatusJSON(statusCode, components.ErrorResponse{
			Code:    statusCode,
			Message: message,
			Error:   message,
		})
		return
	}

	auth.DeleteOrgRoleForUser(user.ID, organization.ID)

	c.Status(http.StatusNoContent)
}
