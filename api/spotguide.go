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
	"net/http"
	"path"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/spotguide"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// GetSpotguide get detailed information about a spotguide
func GetSpotguide(c *gin.Context) {
	log := correlationid.Logger(log, c)

	orgID := auth.GetCurrentOrganization(c.Request).ID

	spotguideName := path.Join(c.Param("owner"), c.Param("name"))
	spotguideVersion := c.Query("version")
	spotguideDetails, err := spotguide.GetSpotguide(orgID, spotguideName, spotguideVersion)
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			c.JSON(http.StatusNotFound, pkgCommon.ErrorResponse{
				Code:    http.StatusNotFound,
				Message: "spotguide not found",
			})
			return
		}
		log.Errorln("error getting spotguide details:", err.Error())
		c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "error getting spotguide details",
		})
		return
	}

	if c.Request.Method == http.MethodHead {
		c.Status(http.StatusOK)
	} else {
		c.JSON(http.StatusOK, spotguideDetails)
	}
}

// GetSpotguides lists all available spotguides
func GetSpotguides(c *gin.Context) {
	log := correlationid.Logger(log, c)

	orgID := auth.GetCurrentOrganization(c.Request).ID

	spotguides, err := spotguide.GetSpotguides(orgID)
	if err != nil {
		log.Errorln("error listing spotguides:", err.Error())
		c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "error listing spotguides",
		})
		return
	}

	c.JSON(http.StatusOK, spotguides)
}

// GetSpotguideIcon returns the icon for the last version of the spotguide
// if not specified otherwise (e.g.: ?version=1.2.3).
func GetSpotguideIcon(c *gin.Context) {
	log := correlationid.Logger(log, c)

	orgID := auth.GetCurrentOrganization(c.Request).ID

	spotguideName := path.Join(c.Param("owner"), c.Param("name"))
	spotguideVersion := c.Query("version")
	spotguideDetails, err := spotguide.GetSpotguide(orgID, spotguideName, spotguideVersion)
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			c.JSON(http.StatusNotFound, pkgCommon.ErrorResponse{
				Code:    http.StatusNotFound,
				Message: "spotguide not found",
			})
			return
		}
		log.Errorln("error getting spotguide details:", err.Error())
		c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "error getting spotguide details",
		})
		return
	}
	body := spotguideDetails.Icon
	if len(body) == 0 {
		c.JSON(http.StatusNotFound, pkgCommon.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "spotguide icon not found",
		})
		return
	}
	// Return the icon SVG data, and mark it as eligible for caching (for 24 hours)
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(http.StatusOK, "image/svg+xml", body)
}

// SyncSpotguidesRateLimit 1 request per 2 minutes
const SyncSpotguidesRateLimit = 1.0 / 60 / 2

// SyncSpotguides synchronizes the spotguide repositories from Github to database
func SyncSpotguides(c *gin.Context) {
	log := correlationid.Logger(log, c)

	orgID := auth.GetCurrentOrganization(c.Request).ID
	userID := auth.GetCurrentUser(c.Request).ID

	err := spotguide.ScrapeSpotguides(orgID, userID)
	if err != nil {
		log.Errorln("failed synchronizing spotguides:", err.Error())
		c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "failed synchronizing spotguides",
		})
		return
	}

	c.Status(http.StatusOK)
}

// LaunchSpotguide creates a spotguide workflow, all secrets, repositories.
func LaunchSpotguide(c *gin.Context) {
	log := correlationid.Logger(log, c)

	var launchRequest spotguide.LaunchRequest
	if err := c.BindJSON(&launchRequest); err != nil {
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "error parsing request",
			Error:   err.Error(),
		})
		return
	}

	orgID := auth.GetCurrentOrganization(c.Request).ID
	userID := auth.GetCurrentUser(c.Request).ID

	err := spotguide.LaunchSpotguide(&launchRequest, c.Request, orgID, userID)
	if err != nil {
		log.Errorf("failed to Launch spotguide %s: %s", launchRequest.RepoFullname(), err.Error())
		c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "error launching spotguide",
			Error:   err.Error(),
		})
		return
	}

	c.Status(http.StatusAccepted)
}
