package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/model"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

//OrganizationMiddleware parses the organization id from the request, queries it from the database and saves it to the current context
//It also checks if the current (calling) user has access to this organization
func OrganizationMiddleware(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": "OrganizationMiddleware"})
	orgidParam := c.Param("orgid")
	orgid, err := strconv.ParseUint(orgidParam, 10, 32)
	if err != nil {
		message := fmt.Sprintf("error parsing organization id: %q", orgidParam)
		log.Info(message)
		c.AbortWithStatusJSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: message,
			Error:   message,
		})
	}

	user := auth.GetCurrentUser(c.Request)
	var organization = auth.Organization{ID: uint(orgid)}

	db := model.GetDB()
	err = db.Model(user).Where(&organization).Related(&organization, "Organizations").Error
	if err != nil {
		message := "error fetching organizations: " + err.Error()
		log.Info(message)
		statusCode := auth.GormErrorToStatusCode(err)
		c.AbortWithStatusJSON(statusCode, components.ErrorResponse{
			Code:    statusCode,
			Message: message,
			Error:   message,
		})
	} else {
		newContext := context.WithValue(c.Request.Context(), auth.CurrentOrganization, &organization)
		c.Request = c.Request.WithContext(newContext)
		c.Next()
	}
}

//GetOrganizations returns all organizations the user belongs to or a specific one from those by id
func GetOrganizations(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": "GetOrganizations"})
	log.Info("Fetching organizations")

	user := auth.GetCurrentUser(c.Request)

	idParam := c.Param("orgid")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if idParam != "" && err != nil {
		message := fmt.Sprintf("error parsing organization id: %s", err)
		log.Info(message)
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: message,
			Error:   message,
		})
		return
	}

	var organization = auth.Organization{ID: uint(id)}
	var organizations []auth.Organization
	db := model.GetDB()
	err = db.Model(user).Where(&organization).Related(&organizations, "Organizations").Error
	if err != nil {
		message := "error fetching organizations"
		log.Info(message + ": " + err.Error())
		statusCode := auth.GormErrorToStatusCode(err)
		c.AbortWithStatusJSON(statusCode, components.ErrorResponse{
			Code:    statusCode,
			Message: message,
			Error:   message,
		})
	} else if id == 0 {
		c.JSON(http.StatusOK, organizations)
	} else if len(organizations) == 1 {
		c.JSON(http.StatusOK, organizations[0])
	} else if len(organizations) > 1 {
		message := fmt.Sprintf("multiple organizations found with id: %q", idParam)
		log.Info(message)
		c.AbortWithStatusJSON(http.StatusConflict, components.ErrorResponse{
			Code:    http.StatusConflict,
			Message: message,
			Error:   message,
		})
	} else {
		message := fmt.Sprintf("organization not found: %q", idParam)
		log.Info(message)
		c.AbortWithStatusJSON(http.StatusNotFound, components.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: message,
			Error:   message,
		})
	}
}

//CreateOrganization creates an organization for the calling user
func CreateOrganization(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": "CreateOrganization"})
	log.Info("Creating organization")

	user, err := auth.GetCurrentUserFromDB(c.Request)
	if err != nil {
		message := "error creating organization"
		log.Info(message + ": " + err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: message,
			Error:   message,
		})
		return
	}

	var name struct {
		Name string `json:"name,omitempty"`
	}

	if err := c.ShouldBindJSON(&name); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Error:   err.Error(),
		})
		return
	}

	organization := auth.Organization{Name: name.Name, Users: []auth.User{*user}}
	db := model.GetDB()
	err = db.Save(&organization).Error
	if err != nil {
		message := "error creating organization: " + err.Error()
		log.Info(message)
		c.AbortWithStatusJSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: message,
			Error:   message,
		})
		return
	}
	c.JSON(http.StatusOK, organization)
}

//DeleteOrganization deletes an organizaion by id
func DeleteOrganization(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": "DeleteOrganization"})
	log.Info("Deleting organization")

	user := auth.GetCurrentUser(c.Request)

	idParam := c.Param("orgid")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		message := fmt.Sprintf("error parsing organization id: %s", err)
		log.Info(message)
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: message,
			Error:   message,
		})
		return
	}

	var organization = auth.Organization{ID: uint(id)}
	db := model.GetDB()
	err = db.Model(user).Where(&organization).Related(&organization, "Organizations").Delete(&organization).Error
	if err != nil {
		message := "error deleting organizations: " + err.Error()
		log.Info(message)
		statusCode := auth.GormErrorToStatusCode(err)
		c.AbortWithStatusJSON(statusCode, components.ErrorResponse{
			Code:    statusCode,
			Message: message,
			Error:   message,
		})
	} else {
		c.Status(http.StatusNoContent)
	}
}
