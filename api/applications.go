package api

import (
	"fmt"
	"github.com/banzaicloud/banzai-types/components"
	ctype "github.com/banzaicloud/banzai-types/components/catalog"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/application"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/model"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
)

// ApplicationDetailsResponse for API
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

// ApplicationListResponse for API TODO move to banzai-types
type ApplicationListResponse struct {
	Id          uint   `json:"id"`
	Name        string `json:"name"`
	ClusterName string `json:"clusterName"`
	ClusterId   uint   `json:"clusterId"`
	Status      string `json:"status"`
	CatalogName string `json:"catalogName"`
	Icon        string `json:"icon"`
}

// DeleteApplications delete application
func DeleteApplications(c *gin.Context) {
	idParam := c.Param("id")
	// Weather to delete cluster as well
	//cluster := c.Query("cluster")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("Invalid id=%q", idParam),
			Error:   err.Error(),
		})
		return
	}
	db := model.GetDB()
	app := &model.Application{
		ID: uint(id),
	}
	var deployments []*model.Deployment
	db.Model(app).Related(&deployments, "Deployments")
	log.Debugf("Associated deployments: %#v", deployments)
	organization := auth.GetCurrentOrganization(c.Request)
	err = db.Model(organization).Related(app).Error
	if err != nil {
		log.Errorf("Error getting application: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error listing clusters",
			Error:   err.Error(),
		})
		return
	}
	app.Deployments = deployments
	application.DeleteApplication(app)
}

// ApplicationDetails get application details
func ApplicationDetails(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("Invalid id=%q", idParam),
			Error:   err.Error(),
		})
		return
	}
	db := model.GetDB()
	application := &model.Application{
		ID: uint(id),
	}
	var deployments []*model.Deployment
	db.Model(application).Related(&deployments, "Deployments")
	log.Debugf("Associated deployments: %#v", deployments)
	organization := auth.GetCurrentOrganization(c.Request)
	err = db.Model(organization).Related(application).Error
	if err != nil {
		log.Errorf("Error getting application: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error listing clusters",
			Error:   err.Error(),
		})
		return
	}
	application.Deployments = deployments
	c.JSON(http.StatusOK, application)
	return
}

// GetApplications gin handler for API
func GetApplications(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": "GetApplications"})
	log.Debug("List applications")

	var applications []model.Application //TODO change this to CommonClusterStatus
	db := model.GetDB()
	organization := auth.GetCurrentOrganization(c.Request)
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

//CreateApplicationRequest  TODO Validate for Cluster xor ClusterId
type CreateApplicationRequest struct {
	Name        string                           `json:"name"`
	CatalogName string                           `json:"catalogName"`
	Cluster     *components.CreateClusterRequest `json:"cluster"`
	ClusterId   uint                             `json:"clusterId"`
	Options     []ctype.ApplicationOptions       `json:"options"`
}

// CreateApplication gin handler for API
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
	orgId := auth.GetCurrentOrganization(c.Request).ID
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
	am := model.Application{
		Name:           createApplicationRequest.Name,
		CatalogName:    createApplicationRequest.CatalogName,
		ClusterID:      commonCluster.GetModel().ID,
		OrganizationId: orgId,
	}
	am.Save()
	go application.CreateApplication(am, createApplicationRequest.Options, commonCluster)
}
