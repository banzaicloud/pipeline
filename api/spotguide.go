package api

import (
	"net/http"
	"strings"

	"github.com/banzaicloud/pipeline/auth"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/spotguide"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// GetSpotguide get detailed information about a spotguide
func GetSpotguide(c *gin.Context) {
	spotguideName := strings.TrimPrefix(c.Param("name"), "/")
	spotguideDetails, err := spotguide.GetSpotguide(spotguideName)
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			c.JSON(http.StatusNotFound, pkgCommon.ErrorResponse{
				Code:    http.StatusNotFound,
				Message: "Spotguide not found",
			})
			return
		}
		log.Errorf("Error getting spotguide details: %s", err.Error())
		c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error getting spotguide details",
		})
		return
	}
	c.JSON(http.StatusOK, spotguideDetails)
}

// GetSpotguides lists all available spotguides
func GetSpotguides(c *gin.Context) {
	spotguides, err := spotguide.GetSpotguides()
	if err != nil {
		log.Errorf("Error listing spotguides: %s", err.Error())
		c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error listing spotguides",
		})
		return
	}

	c.JSON(http.StatusOK, spotguides)
}

// SyncSpotguides synchronizes the spotguide repositories from Github to database
func SyncSpotguides(c *gin.Context) {
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
	var launchRequest spotguide.LaunchRequest
	if err := c.BindJSON(&launchRequest); err != nil {
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}
	org := auth.GetCurrentOrganization(c.Request)
	user := auth.GetCurrentUser(c.Request)
	spotguide.LaunchSpotguide(&launchRequest, org.ID, user.ID)
	c.Status(http.StatusAccepted)
}
