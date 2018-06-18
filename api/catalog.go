package api

import (
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/catalog"
	"github.com/gin-gonic/gin"
	"net/http"
)

// CatalogDetails get detailed information about a catalog
func CatalogDetails(c *gin.Context) {
	organization := auth.GetCurrentOrganization(c.Request)
	env := catalog.GenerateCatalogEnv(organization.Name)
	chartName := c.Param("name")
	log.Debugln("chartName:", chartName)
	chartDetails, err := catalog.GetCatalogDetails(env, chartName)
	if err != nil {
		log.Errorf("Error parsing request: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "error parsing request",
			Error:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, chartDetails)
	return
}

// GetCatalogs List available Catalogs
func GetCatalogs(c *gin.Context) {
	organization := auth.GetCurrentOrganization(c.Request)
	env := catalog.GenerateCatalogEnv(organization.Name)
	// Initialise filter type
	filter := ParseField(c)
	// Filter for organisation
	filter["organization_id"] = organization.ID

	chartName := c.Param("name")
	log.Debugln("chartName:", chartName)

	chartVersion := c.Param("version")
	log.Debugln("version:", chartVersion)
	catalogs, err := catalog.ListCatalogs(env, chartName, chartVersion, "")
	if err != nil {
		log.Error("Empty cluster list")
		c.JSON(http.StatusNotFound, components.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "Catalogs not found",
			Error:   "Catalogs not found",
		})
		return
	}

	c.JSON(http.StatusOK, catalogs)
}
