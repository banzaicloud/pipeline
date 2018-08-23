package api

import (
	"net/http"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/catalog"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/gin-gonic/gin"
)

// CatalogDetails get detailed information about a catalog
func CatalogDetails(c *gin.Context) {
	organization := auth.GetCurrentOrganization(c.Request)
	env := catalog.GenerateCatalogEnv(organization.Name)
	catalogName := c.Param("name")
	catalogDetails, err := catalog.GetCatalogDetails(env, catalogName)
	if err != nil {
		log.Errorf("Error getting catalog details: %s", err.Error())
		c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error getting catalog details",
		})
		return
	}

	if catalogDetails == nil {
		c.JSON(http.StatusNotFound, pkgCommon.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "Catalog not found",
		})
		return
	}

	c.JSON(http.StatusOK, catalogDetails)
}

// GetCatalogs List available Catalogs
func GetCatalogs(c *gin.Context) {
	organization := auth.GetCurrentOrganization(c.Request)
	env := catalog.GenerateCatalogEnv(organization.Name)

	catalogName := c.Query("name")
	catalogVersion := c.Query("version")

	catalogs, err := catalog.ListCatalogs(env, catalogName, catalogVersion, "")
	if err != nil {
		log.Errorf("Error listing catalogs: %s", err.Error())
		c.JSON(http.StatusNotFound, pkgCommon.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "Catalogs not found",
		})
		return
	}

	c.JSON(http.StatusOK, catalogs)
}

// UpdateCatalogs will update helm repository under catalog
func UpdateCatalogs(c *gin.Context) {
	organization := auth.GetCurrentOrganization(c.Request)
	env := catalog.GenerateCatalogEnv(organization.Name)
	err := catalog.CatalogUpdate(env)
	if err != nil {
		c.JSON(http.StatusNotFound, pkgCommon.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "Catalog update failed",
		})
		return
	}

	c.Status(http.StatusAccepted)
}
