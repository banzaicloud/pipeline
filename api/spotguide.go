package api

import (
	"net/http"
	"strings"

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

	spotguideName := strings.TrimPrefix(c.Param("name"), "/")
	spotguideDetails, err := spotguide.GetSpotguide(spotguideName)
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			c.JSON(http.StatusNotFound, pkgCommon.ErrorResponse{
				Code:    http.StatusNotFound,
				Message: "spotguide not found",
			})
			return
		}
		log.Errorln("Error getting spotguide details:", err.Error())
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

	spotguides, err := spotguide.GetSpotguides()
	if err != nil {
		log.Errorln("Error listing spotguides:", err.Error())
		c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "error listing spotguides",
		})
		return
	}

	c.JSON(http.StatusOK, spotguides)
}

// SyncSpotguides synchronizes the spotguide repositories from Github to database
func SyncSpotguides(c *gin.Context) {
	log := correlationid.Logger(log, c)

	go func() {
		err := spotguide.ScrapeSpotguides()
		if err != nil {
			log.Errorln("Failed synchronizing spotguides:", err.Error())
		}
	}()

	c.Status(http.StatusAccepted)
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

	org := auth.GetCurrentOrganization(c.Request)
	user := auth.GetCurrentUser(c.Request)

	err := spotguide.LaunchSpotguide(&launchRequest, org.ID, user.ID)
	if err != nil {
		log.Errorln("Failed to Launch spotguide:", err.Error())
		c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "error launching spotguide",
		})
		return
	}

	c.Status(http.StatusAccepted)
}
