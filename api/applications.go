package api

import (
	"github.com/banzaicloud/banzai-types/components"
	htype "github.com/banzaicloud/banzai-types/components/helm"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net/http"
)

func GetApplications(c *gin.Context) {
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

//Create Application
// Validate for Cluster xor ClusterId
type CreateApplicationRequest struct {
	Name        string                                `json:"name"`
	Cluster     components.CreateClusterRequest       `json:"cluster"`
	ClusterId   uint                                  `json:"cluster_id"`
	Deployments []htype.CreateUpdateDeploymentRequest `json:"deployments"`
}

func CreateApplication(c *gin.Context) {
	// 1. Create Pending applications in database
	log := logger.WithFields(logrus.Fields{"tag": constants.TagCreateCluster})
	//TODO refactor logging here

	log.Info("Cluster creation stared")

	log.Debug("Bind json into CreateClusterRequest struct")
	// bind request body to struct
	var createApplicationRequest CreateApplicationRequest
	if err := c.BindJSON(&createApplicationRequest); err != nil {
		log.Error(errors.Wrap(err, "Error parsing request"))
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}
	// Create cluster TODO check if async enough
	CreateCluster(c, &createApplicationRequest.Cluster)

	cluster.CreateApplication()
	// 5. Deploy applications

}
