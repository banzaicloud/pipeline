package api

import (
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/application"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/catalog"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/model"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net/http"
)

type ApplicationDetailsResponse struct {
	Name        string `json:"name"`
	ClusterName string `json:"clusterName"`
	ClusterId   int    `json:"clusterId"`
	Status      string
	Icon        string
	Deployments []string
	Error       string
	//Spotguide
}

type ApplicationListResponse struct {
	Id          uint   `json:"id"`
	Name        string `json:"name"`
	ClusterName string `json:"clusterName"`
	ClusterId   uint   `json:"clusterId"`
	Status      string `json:"status"`
	CatalogName string `json:"catalogName"`
	Icon        string `json:"icon"`
}

func GetApplications(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": "GetApplications"})
	log.Debug("List applications")

	var applications []model.ApplicationModel //TODO change this to CommonClusterStatus
	db := model.GetDB()
	organization := auth.GetCurrentOrganization(c.Request)
	organization.Name = ""
	err := db.Model(organization).Related(&applications).Error
	if err != nil {
		log.Errorf("Error listing clusters: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error listing clusters",
			Error:   err.Error(),
		})
		return
	}
	response := make([]ApplicationListResponse, 0)
	for _, app := range applications {
		log.Debugf("Apps: %#v", applications)
		item := ApplicationListResponse{
			Id:          app.ID,
			Name:        app.Name,
			CatalogName: app.CatalogName,
			Icon:        app.Icon,
			ClusterId:   app.GetCluster().ID,
			ClusterName: app.GetCluster().Name,
			Status:      app.Status,
		}
		response = append(response, item)
	}
	c.JSON(http.StatusOK, response)
	return
}

func ApplicationDetails() {

}

//Create Application
// Validate for Cluster xor ClusterId
type CreateApplicationRequest struct {
	Name        string                           `json:"name"`
	CatalogName string                           `json:"catalogName"`
	Cluster     *components.CreateClusterRequest `json:"cluster"`
	ClusterId   uint                             `json:"clusterId"`
	Options     []catalog.ApplicationOptions     `json:"options"`
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
	orgId := c.Request.Context().Value(auth.CurrentOrganization).(*auth.Organization).ID
	var commonCluster cluster.CommonCluster
	// Create new cluster
	if createApplicationRequest.Cluster != nil {
		commonCluster = CreateCluster(c, createApplicationRequest.Cluster)
		// Support existing cluster
	} else {
		filter := make(map[string]interface{})
		filter["organization_id"] = orgId
		filter["id"] = createApplicationRequest.ClusterId
		var ok bool
		commonCluster, ok = GetCommonClusterFromFilter(c, filter)
		if !ok {
			return
		}
	}
	am := model.ApplicationModel{
		Name:           createApplicationRequest.Name,
		CatalogName:    createApplicationRequest.CatalogName,
		ClusterID:      commonCluster.GetModel().ID,
		OrganizationId: orgId,
	}
	am.Save()
	go application.CreateApplication(am, createApplicationRequest.Options, commonCluster)
}
