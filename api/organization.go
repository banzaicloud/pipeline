package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/model"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

//OrganizationMiddleware parses the organization id from the request, queries it from the database and saves it to the current context
func OrganizationMiddleware(c *gin.Context) {
	orgidParam := c.Param("orgid")
	orgid, err := strconv.ParseUint(orgidParam, 10, 32)
	if err != nil {
		message := "Error parsing organization id"
		log.Info(message, ": ", err)
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: message,
			Error:   message,
		})
	}

	user := auth.GetCurrentUser(c.Request)
	var organization = auth.Organization{ID: uint(orgid)}
	var organizations []auth.Organization

	db := model.GetDB()
	err = db.Model(user).Where(&organization).Related(&organizations, "Organizations").Error
	if err != nil {
		message := "Error listing organizations"
		log.Info(message, ": ", err)
		c.JSON(http.StatusInternalServerError, components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: message,
			Error:   message,
		})
		return
	}

	if len(organizations) != 1 {
		message := "Organization not found"
		log.Info(message)
		c.JSON(http.StatusNotFound, components.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: message,
			Error:   message,
		})
		return
	}
	newContext := context.WithValue(c.Request.Context(), auth.CurrentOrganization, &organizations[0])
	c.Request = c.Request.WithContext(newContext)
	c.Next()
}

//GetOrganizations returns all organizations the user belongs to or a specific one from those by id
func GetOrganizations(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": "GetOrganizations"})
	log.Info("Fetching organizations")

	user := auth.GetCurrentUser(c.Request)

	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if idParam != "" && err != nil {
		message := "Error parsing organization id"
		log.Info(message, ": ", err)
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
		message := "Error listing organizations"
		log.Info(message, ": ", err)
		c.JSON(http.StatusInternalServerError, components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: message,
			Error:   message,
		})
	} else if id == 0 {
		c.JSON(http.StatusOK, organizations)
	} else if len(organizations) != 0 {
		c.JSON(http.StatusOK, organizations[0])
	} else {
		message := "Organization not found"
		log.Info(message)
		c.JSON(http.StatusNotFound, components.ErrorResponse{
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
		message := "Error creating organization"
		log.Info(message, ": ", err)
		c.JSON(http.StatusInternalServerError, components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: message,
			Error:   message,
		})
		return
	}

	var name struct {
		Name string
	}

	if err := c.ShouldBindJSON(&name); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	organization := auth.Organization{Name: name.Name, Users: []auth.User{*user}}
	db := model.GetDB()
	err = db.Save(&organization).Error
	if err != nil {
		message := "Error creating organization"
		log.Info(message, ": ", err)
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: message,
			Error:   message,
		})
		return
	}
	c.JSON(http.StatusOK, organization)
}
