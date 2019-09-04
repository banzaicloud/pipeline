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
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
	"github.com/banzaicloud/pipeline/pkg/common"
	"github.com/gin-gonic/gin"
)

// OrganizationMiddleware parses the organization id from the request,
// queries it from the database and saves it to the current context.
func OrganizationMiddleware(c *gin.Context) {
	orgidParam := c.Param("orgid")
	orgid, err := strconv.ParseUint(orgidParam, 10, 32)
	if err != nil {
		message := fmt.Sprintf("error parsing organization id: %q", orgidParam)
		log.Info(message)
		c.AbortWithStatusJSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: message,
			Error:   message,
		})
		return
	}

	organization := &auth.Organization{ID: uint(orgid)}

	db := config.DB()
	err = db.Where(organization).Find(organization).Error
	if err != nil {
		message := "error fetching organizations: " + err.Error()
		log.Info(message)
		statusCode := auth.GormErrorToStatusCode(err)
		c.AbortWithStatusJSON(statusCode, common.ErrorResponse{
			Code:    statusCode,
			Message: message,
			Error:   message,
		})
	} else {
		newContext := context.WithValue(c.Request.Context(), auth.CurrentOrganization, organization)
		c.Request = c.Request.WithContext(newContext)
		c.Next()
	}
}

// OrganizationAPI implements organization functions.
type OrganizationAPI struct {
	orgImporter *auth.OrgImporter
}

// NewOrganizationAPI returns a new OrganizationAPI instance.
func NewOrganizationAPI(orgImporter *auth.OrgImporter) *OrganizationAPI {
	return &OrganizationAPI{
		orgImporter: orgImporter,
	}
}

// GetOrganizations returns all organizations the user belongs to or a specific one from those by id.
func (a *OrganizationAPI) GetOrganizations(c *gin.Context) {
	log.Info("Fetching organizations")

	user := auth.GetCurrentUser(c.Request)

	idParam := c.Param("orgid")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if idParam != "" && err != nil {
		message := fmt.Sprintf("error parsing organization id: %s", err)
		log.Info(message)
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: message,
			Error:   message,
		})
		return
	}

	var organization = auth.Organization{ID: uint(id)}
	var organizations []auth.Organization

	db := config.DB()

	// Virtual users can list only the organization they are belonging to
	if user.Virtual {
		organization.Name = auth.GetOrgNameFromVirtualUser(user.Login)
		err = db.Where(&organization).Find(&organizations).Error
	} else {
		err = db.Model(user).Where(&organization).Related(&organizations, "Organizations").Error
	}

	if err != nil {
		message := "error fetching organizations"
		log.Info(message + ": " + err.Error())
		statusCode := auth.GormErrorToStatusCode(err)
		c.AbortWithStatusJSON(statusCode, common.ErrorResponse{
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
		c.AbortWithStatusJSON(http.StatusConflict, common.ErrorResponse{
			Code:    http.StatusConflict,
			Message: message,
			Error:   message,
		})
	} else {
		message := fmt.Sprintf("organization not found: %q", idParam)
		log.Info(message)
		c.AbortWithStatusJSON(http.StatusNotFound, common.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: message,
			Error:   message,
		})
	}
}

// SyncOrganizations synchronizes github organizations.
func (a *OrganizationAPI) SyncOrganizations(c *gin.Context) {
	logger := correlationid.Logger(log, c)

	logger.Info("synchronizing organizations")

	user := auth.GetCurrentUser(c.Request)

	err := auth.SyncOrgsForUser(a.orgImporter, user, c.Request)
	if err != nil {
		errorHandler.Handle(err)

		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "synchronization failed",
			Error:   err.Error(),
		})

		return
	}

	c.Status(http.StatusOK)
}

// DeleteOrganization deletes an organization by id.
func (a *OrganizationAPI) DeleteOrganization(c *gin.Context) {
	log.Info("Deleting organization")

	idParam := c.Param("orgid")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		message := fmt.Sprintf("error parsing organization id: %s", err)
		log.Info(message)
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: message,
			Error:   message,
		})
		return
	}

	user := auth.GetCurrentUser(c.Request)
	organization, err := auth.GetOrganizationById(uint(id))
	deleteName := organization.Name

	err = deleteOrgFromDB(organization, user)
	if err != nil {
		message := "error deleting organizations: " + err.Error()
		log.Info(message)
		statusCode := auth.GormErrorToStatusCode(err)
		c.AbortWithStatusJSON(statusCode, common.ErrorResponse{
			Code:    statusCode,
			Message: message,
			Error:   message,
		})
	} else {

		log.Infof("Clean org's statestore folder %s", deleteName)
		if err := cluster.CleanHelmFolder(deleteName); err != nil {
			log.Errorf("Statestore cleaning failed: %s", err.Error())
		} else {
			log.Info("Org's statestore folder cleaned")
		}

		c.Status(http.StatusNoContent)
	}
}

func deleteOrgFromDB(organization *auth.Organization, user *auth.User) error {
	tx := config.DB().Begin()
	err := tx.Error
	if err != nil {
		tx.Rollback()
		return err
	}
	err = tx.Model(user).Where(organization).Related(organization, "Organizations").Delete(organization).Error
	if err != nil {
		tx.Rollback()
		return err
	}
	err = tx.Model(user).Association("Organizations").Delete(organization).Error
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}
