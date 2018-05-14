package api

import (
	"github.com/banzaicloud/banzai-types/components"
	htype "github.com/banzaicloud/banzai-types/components/helm"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/model"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
)

//TODO check if we need transformation
type Catalog struct {
}

func GetCatalogDetails(c *gin.Context) {

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

func ListCatalogs(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": "ListCatalogs"})
	log.Info("Get helm repository charts")

	var query ChartQuery
	err := c.BindQuery(&query)
	if err != nil {
		log.Errorf("Error parsing request: %s", err.Error())
		c.JSON(http.StatusBadRequest, htype.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "error parsing request",
			Error:   err.Error(),
		})
		return
	}
	log.Info(query)
	response, err := helm.ListCatalogs(query.Name, query.Version, query.Keyword)
	if err != nil {
		log.Error("Error during get helm repo chart list.", err.Error())
		c.JSON(http.StatusBadRequest, htype.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error listing helm repo charts",
			Error:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, response)
	return
}

func CreateCatalog(c *gin.Context) {

}

//Get
func GetCatalogs(c *gin.Context) {
	// Initialise filter type
	filter := ParseField(c)
	// Filter for organisation
	filter["organization_id"] = c.Request.Context().Value(auth.CurrentOrganization).(*auth.Organization).ID
	//catalogs := make([]model.CatalogModel, 0)
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
