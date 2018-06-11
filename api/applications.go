package api

import (
	"fmt"
	"github.com/banzaicloud/banzai-types/components"
	bApplication "github.com/banzaicloud/banzai-types/components/application"
	"github.com/banzaicloud/pipeline/application"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/model"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"net/http"
	"strconv"
)

func getApplicationFromRequest(c *gin.Context) (*model.Application, bool) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("Invalid id=%q", idParam),
			Error:   err.Error(),
		})
		return nil, false
	}
	db := model.GetDB()
	application := &model.Application{
		ID: uint(id),
	}
	organization := auth.GetCurrentOrganization(c.Request)
	err = db.Model(organization).Related(application).Error
	if err != nil {
		log.Errorf("Error getting application: %s", err.Error())
		statusCode := auth.GormErrorToStatusCode(err)
		c.JSON(statusCode, components.ErrorResponse{
			Code:    statusCode,
			Message: "Error listing clusters",
			Error:   err.Error(),
		})
		return nil, false
	}
	var deployments []*model.Deployment
	db.Model(application).Related(&deployments, "Deployments")
	log.Debugf("Associated deployments: %#v", deployments)
	application.Deployments = deployments
	return application, true
}

func getCluster(app *model.Application, c *gin.Context) (*model.ClusterModel, bool) {
	clusterModel, err := app.GetCluster()
	if err != nil {
		log.Errorf("Error getting cluster: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting cluster",
			Error:   err.Error(),
		})
		return nil, false
	}
	return clusterModel, true
}

// DeleteApplications delete application
func DeleteApplications(c *gin.Context) {

	forceParam := c.DefaultQuery("force", "false")
	force, err := strconv.ParseBool(forceParam)
	if err != nil {
		force = false
	}

	app, ok := getApplicationFromRequest(c)
	if !ok {
		return
	}
	clusterModel, ok := getCluster(app, c)
	if !ok {
		return
	}
	commonCluster, err := cluster.GetCommonClusterFromModel(clusterModel)
	if err != nil {
		log.Errorf("Error getting cluster: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting cluster",
			Error:   err.Error(),
		})
		return
	}
	config, err := commonCluster.GetK8sConfig()
	if err != nil && !force {
		log.Errorf("Error getting cluster config: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting cluster config",
			Error:   err.Error(),
		})
		return
	}

	if err := application.DeleteApplication(app, config, force); err != nil {
		log.Errorf("Error during deleting application: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during deleting application",
			Error:   err.Error(),
		})
		return
	}

	app.Delete()
	c.Status(http.StatusOK)

}

// ApplicationDetails get application details
func ApplicationDetails(c *gin.Context) {
	application, ok := getApplicationFromRequest(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, application)
	return
}

// GetApplications gin handler for API
func GetApplications(c *gin.Context) {
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
	response := make([]bApplication.ListResponse, 0)
	log.Debugf("Apps: %#v", applications)
	for _, app := range applications {
		//We silently fail on GetCluster
		clusterModel, err := app.GetCluster()
		if err != nil {
			continue
		}
		item := bApplication.ListResponse{
			Id:            app.ID,
			Name:          app.Name,
			CatalogName:   app.CatalogName,
			Icon:          app.Icon,
			ClusterId:     clusterModel.ID,
			ClusterName:   clusterModel.Name,
			Status:        app.Status,
			StatusMessage: app.Message,
		}
		response = append(response, item)
	}
	c.JSON(http.StatusOK, response)
	return
}

// CreateApplication gin handler for API
func CreateApplication(c *gin.Context) {
	var createApplicationRequest bApplication.CreateRequest
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
		// Support existing cluster
		var err *components.ErrorResponse
		commonCluster, err = CreateCluster(createApplicationRequest.Cluster, orgId)
		if err != nil {
			c.JSON(err.Code, err)
			return
		}
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
		Status:         application.CREATING,
	}
	am.Save()

	c.JSON(http.StatusAccepted, bApplication.CreateResponse{
		Name:      createApplicationRequest.Name,
		Id:        am.ID,
		ClusterId: commonCluster.GetModel().ID,
	})

	go application.CreateApplication(am, createApplicationRequest.Options, commonCluster)
}
