package api

import (
	"github.com/banzaicloud/banzai-types/components"
	htype "github.com/banzaicloud/banzai-types/components/helm"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/model"
	"github.com/gin-gonic/gin"
	"net/http"
)

//TODO check if we need transformation
type Catalog struct {
}

func GetCatalogDetails(c *gin.Context) {
	//Infromation about your running catalog
}

type CreateCatalogsRequests struct {
}

func CatalogDetails(c *gin.Context) {
	chartName := c.Param("name")
	log.Debugln("chartName:", chartName)
	chartDetails, err := helm.GetCatalogDetails(chartName)
	if err != nil {
		log.Errorf("Error parsing request: %s", err.Error())
		c.JSON(http.StatusBadRequest, htype.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "error parsing request",
			Error:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, chartDetails)
	return
}

//Get
func GetCatalogs(c *gin.Context) {
	// Initialise filter type
	filter := ParseField(c)
	// Filter for organisation
	filter["organization_id"] = c.Request.Context().Value(auth.CurrentOrganization).(*auth.Organization).ID
	//catalogs := make([]model.ApplicationModel, 0)
	catalogs, err := model.QueryCatalog(filter)
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
