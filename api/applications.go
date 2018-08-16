package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/banzaicloud/pipeline/application"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	"github.com/banzaicloud/pipeline/model"
	pkgApplication "github.com/banzaicloud/pipeline/pkg/application"
	pkgCatalog "github.com/banzaicloud/pipeline/pkg/catalog"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

func getApplicationFromRequest(c *gin.Context) (*model.Application, bool) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("Invalid id=%q", idParam),
			Error:   err.Error(),
		})
		return nil, false
	}
	db := config.DB()
	application := &model.Application{
		ID: uint(id),
	}
	organization := auth.GetCurrentOrganization(c.Request)
	err = db.Model(organization).Related(application).Error
	if err != nil {
		log.Errorf("Error getting application: %s", err.Error())
		statusCode := auth.GormErrorToStatusCode(err)
		c.JSON(statusCode, pkgCommon.ErrorResponse{
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
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
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
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting cluster",
			Error:   err.Error(),
		})
		return
	}
	config, err := commonCluster.GetK8sConfig()
	if err != nil && !force {
		log.Errorf("Error getting cluster config: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting cluster config",
			Error:   err.Error(),
		})
		return
	}

	if err := application.DeleteApplication(app, config, force); err != nil {
		log.Errorf("Error during deleting application: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
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

// GetApplicationsByCluster gin handler for API
func GetApplicationsByCluster(c *gin.Context) {
	log.Debug("List applications by cluster")

	commonCluster, ok := GetCommonClusterFromRequest(c)
	if ok != true {
		return
	}

	clusterId := commonCluster.GetID()
	log.Infof("find applications by cluster id [%d]", clusterId)
	applications, err := application.FindApplicationsByCluster(clusterId)
	if err != nil {
		log.Errorf("error during getting applications")
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "error during getting applications",
		})
		return
	}

	c.JSON(http.StatusOK, applications)

}

// GetApplications gin handler for API
func GetApplications(c *gin.Context) {
	log.Debug("List applications")

	var applications []model.Application //TODO change this to CommonClusterStatus
	db := config.DB()
	organization := auth.GetCurrentOrganization(c.Request)
	err := db.Model(organization).Related(&applications).Error
	if err != nil {
		log.Errorf("Error listing clusters: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error listing clusters",
			Error:   err.Error(),
		})
		return
	}
	response := make([]pkgApplication.ListResponse, 0)
	log.Debugf("Apps: %#v", applications)
	for _, app := range applications {
		//We silently fail on GetCluster
		clusterModel, err := app.GetCluster()
		if err != nil {
			continue
		}

		item := pkgApplication.ListResponse{
			Id:                app.ID,
			Name:              app.Name,
			ClusterName:       clusterModel.Name,
			ClusterId:         clusterModel.ID,
			Status:            app.Status,
			CatalogName:       app.CatalogName,
			Icon:              app.Icon,
			StatusMessage:     app.Message,
			CreatorBaseFields: *cluster.NewCreatorBaseFields(app.CreatedAt, app.CreatedBy),
		}
		response = append(response, item)
	}
	c.JSON(http.StatusOK, response)
	return
}

// ApplicationPostHook describes create application posthook
type ApplicationPostHook struct {
	am     *model.Application
	option []pkgCatalog.ApplicationOptions
}

// Do updates application in DB and call create application function
func (c *ApplicationPostHook) Do(commonCluster cluster.CommonCluster) error {
	c.Save(commonCluster.GetID())
	return application.CreateApplication(c.am, c.option, commonCluster)
}

func (c *ApplicationPostHook) Error(commonCluster cluster.CommonCluster, err error) {
	c.am.ClusterID = commonCluster.GetID()
	c.am.Update(model.Application{Status: application.FAILED, Message: err.Error()})
}

// Save application to DB
func (c *ApplicationPostHook) Save(clusterId uint) {
	c.am.ClusterID = clusterId
	c.am.Save()
}

// GetID returns application identifier
func (c *ApplicationPostHook) GetID() uint {
	return c.am.ID
}

// CreateApplication gin handler for API
func CreateApplication(c *gin.Context) {
	var createApplicationRequest pkgApplication.CreateRequest
	if err := c.ShouldBindJSON(&createApplicationRequest); err != nil {
		log.Error(errors.Wrap(err, "Error parsing request"))
		c.AbortWithStatusJSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}

	orgId := auth.GetCurrentOrganization(c.Request).ID
	userId := auth.GetCurrentUser(c.Request).ID

	postFunction := &ApplicationPostHook{
		am: &model.Application{
			Name:           createApplicationRequest.Name,
			CatalogName:    createApplicationRequest.CatalogName,
			OrganizationId: orgId,
			Status:         application.CREATING,
			CreatedBy:      userId,
		},
		option: createApplicationRequest.Options,
	}

	var commonCluster cluster.CommonCluster
	// Create new cluster
	if createApplicationRequest.Cluster != nil {
		// Support existing cluster
		var err *pkgCommon.ErrorResponse
		ctx := ginutils.Context(context.Background(), c)
		commonCluster, err = CreateCluster(ctx, createApplicationRequest.Cluster, orgId, userId, []cluster.PostFunctioner{postFunction})
		if err != nil {
			c.JSON(err.Code, err)
			return
		}

		postFunction.Save(commonCluster.GetID())

	} else {
		filter := make(map[string]interface{})
		filter["organization_id"] = orgId
		filter["id"] = createApplicationRequest.ClusterId
		var ok bool
		commonCluster, ok = GetCommonClusterFromFilter(c, filter)
		if !ok {
			return
		}

		postFunction.Save(commonCluster.GetID())

		go postFunction.Do(commonCluster)
	}

	c.JSON(http.StatusAccepted, pkgApplication.CreateResponse{
		Name:      createApplicationRequest.Name,
		Id:        postFunction.GetID(),
		ClusterId: commonCluster.GetID(),
	})

}
