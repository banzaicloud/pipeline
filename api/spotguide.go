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

	"emperror.dev/emperror"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/api/middleware"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/spotguide"
)

type SpotguideAPI struct {
	logger       logrus.FieldLogger
	errorHandler emperror.Handler
	spotguide    *spotguide.SpotguideManager
}

// GetSpotguide get detailed information about a spotguide
func (s *SpotguideAPI) GetSpotguide(c *gin.Context) {
	log := correlationid.Logger(log, c)

	orgID := auth.GetCurrentOrganization(c.Request).ID

	spotguideName := path.Join(c.Param("owner"), c.Param("name"))
	spotguideVersion := c.Query("version")
	spotguideDetails, err := s.spotguide.GetSpotguide(orgID, spotguideName, spotguideVersion)
	if err != nil {
		if gorm.IsRecordNotFoundError(errors.Cause(err)) {
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
func (s *SpotguideAPI) GetSpotguides(c *gin.Context) {
	log := correlationid.Logger(log, c)

	orgID := auth.GetCurrentOrganization(c.Request).ID

	spotguides, err := s.spotguide.GetSpotguides(orgID)
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
func (s *SpotguideAPI) GetSpotguideIcon(c *gin.Context) {
	log := correlationid.Logger(log, c)

	orgID := auth.GetCurrentOrganization(c.Request).ID

	spotguideName := path.Join(c.Param("owner"), c.Param("name"))
	spotguideVersion := c.Query("version")
	spotguideDetails, err := s.spotguide.GetSpotguide(orgID, spotguideName, spotguideVersion)
	if err != nil {
		if gorm.IsRecordNotFoundError(errors.Cause(err)) {
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

	// Return the icon SVG data, and mark it as eligible for caching (for 24 hours)
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(http.StatusOK, "image/svg+xml", spotguideDetails.Icon)
}

// SyncSpotguidesRateLimit 1 request per 2 minutes
const SyncSpotguidesRateLimit = 1.0 / 60 / 2

// SyncSpotguides synchronizes the spotguide repositories from Github to database
func (s *SpotguideAPI) SyncSpotguides(c *gin.Context) {
	log := correlationid.Logger(log, c)

	orgID := auth.GetCurrentOrganization(c.Request).ID
	userID := auth.GetCurrentUser(c.Request).ID

	err := s.spotguide.ScrapeSpotguides(orgID, userID)
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
func (s *SpotguideAPI) LaunchSpotguide(c *gin.Context) {

	var launchRequest spotguide.LaunchRequest
	if err := c.BindJSON(&launchRequest); err != nil {
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "error parsing request",
			Error:   err.Error(),
		})
		return
	}

	org := auth.GetCurrentOrganization(c.Request)
	user := auth.GetCurrentUser(c.Request)

	err := s.spotguide.LaunchSpotguide(&launchRequest, org, user)
	if err != nil {
		s.errorHandler.Handle(emperror.WrapWith(err, "failed to launch spotguide", "spotguide", launchRequest.RepoFullname()))
		c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "error launching spotguide",
			Error:   err.Error(),
		})
		return
	}

	c.Status(http.StatusAccepted)
}

func NewSpotguideAPI(logger logrus.FieldLogger, errorHandler emperror.Handler, spotguideManager *spotguide.SpotguideManager) *SpotguideAPI {
	return &SpotguideAPI{
		logger:       logger,
		errorHandler: errorHandler,
		spotguide:    spotguideManager,
	}
}

func (s *SpotguideAPI) Install(spotguides *gin.RouterGroup) {
	spotguides.GET("", s.GetSpotguides)
	spotguides.PUT("", middleware.NewRateLimiterByOrgID(SyncSpotguidesRateLimit), s.SyncSpotguides)
	spotguides.POST("", s.LaunchSpotguide)
	// Spotguide name may contain '/'s so we have to use :owner/:name
	spotguides.GET("/:owner/:name", s.GetSpotguide)
	spotguides.HEAD("/:owner/:name", s.GetSpotguide)
	spotguides.GET("/:owner/:name/icon", s.GetSpotguideIcon)
}
