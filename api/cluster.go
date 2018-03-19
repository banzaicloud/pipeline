package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/model"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// TODO see who will win
var logger *logrus.Logger
var log *logrus.Entry

// Simple init for logging
func init() {
	logger = config.Logger()
	log = logger.WithFields(logrus.Fields{"tag": "Cluster"})
}

//ParseField is to restrict other query TODO investigate to just pass the hasmap
func ParseField(c *gin.Context) map[string]interface{} {
	value := c.Param("id")
	field := c.DefaultQuery("field", "id")
	return map[string]interface{}{field: value}
}

func UpdateMonitoring(c *gin.Context) {
	cluster.UpdatePrometheus()
	c.String(http.StatusOK, "OK")
	return
}

// GetCommonClusterFromRequest just a simple getter to build commonCluster object this handles error messages directly
func GetCommonClusterFromRequest(c *gin.Context) (cluster.CommonCluster, bool) {
	filter := ParseField(c)

	// Filter for organisation
	filter["organization_id"] = c.Request.Context().Value(auth.CurrentOrganization).(*auth.Organization).ID

	//TODO check gorm error
	modelCluster, err := model.QueryCluster(filter)
	if err != nil {
		log.Errorf("Cluster not found: %s", err.Error())
		c.JSON(http.StatusNotFound, components.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "Cluster not found",
			Error:   err.Error(),
		})
		return nil, false
	}
	commonCLuster, err := cluster.GetCommonClusterFromModel(modelCluster)
	if err != nil {
		log.Errorf("GetCommonClusterFromModel failed: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return nil, false
	}
	return commonCLuster, true
}

// CreateCluster creates a K8S cluster in the cloud
func CreateCluster(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": constants.TagCreateCluster})
	//TODO refactor logging here

	log.Info("Cluster creation stared")

	log.Debug("Bind json into CreateClusterRequest struct")
	// bind request body to struct
	var createClusterRequest components.CreateClusterRequest
	if err := c.BindJSON(&createClusterRequest); err != nil {
		log.Error(errors.Wrap(err, "Error parsing request"))
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}
	log.Debug("Parsing request succeeded")

	log.Info("Searching entry with name: ", createClusterRequest.Name)

	// check exists cluster name
	var existingCluster model.ClusterModel
	database := model.GetDB()
	database.Raw("SELECT * FROM "+model.ClusterModel.TableName(existingCluster)+" WHERE name = ?;",
		createClusterRequest.Name).Scan(&existingCluster)

	////TODO check if error handling is enough
	//existingCluster, err := model.QueryCluster(filter)
	//if err != nil {
	//	log.Error(err)
	//	c.JSON(http.StatusBadRequest, ErrorResponse{
	//		Code:    http.StatusBadRequest,
	//		Message: "Error parsing request",
	//		Error:   err.Error(),
	//	})
	//	return
	//}

	if existingCluster.ID != 0 {
		// duplicated entry
		err := fmt.Errorf("duplicate entry: %s", existingCluster.Name)
		log.Error(err)
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Error:   err.Error(),
		})
		return
	}

	log.Info("Creating new entry with cloud type: ", createClusterRequest.Cloud)

	var commonCluster cluster.CommonCluster

	// TODO check validation
	commonCluster, err := cluster.CreateCommonClusterFromRequest(&createClusterRequest)
	if err != nil {
		log.Errorf("Error during creating common cluster model: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Error:   err.Error(),
		})
		return
	}
	// This is the common part of cluster flow
	organizationID := auth.GetCurrentOrganization(c.Request).IDString()
	// Create cluster
	err = commonCluster.CreateCluster(organizationID)
	if err != nil {
		log.Errorf("Error during cluster creation: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Error:   err.Error(),
		})
		return
	}

	// Persist the cluster in Database
	err = commonCluster.Persist()
	if err != nil {
		log.Errorf("Error persisting cluster in database: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Error:   err.Error(),
		})
		return
	}

	// Apply PostHooks
	// These are hardcoded posthooks maybe we will want a bit more dynamic
	postHookFunctions := []func(commonCluster cluster.CommonCluster){
		cluster.PersistKubernetesKeys,
		cluster.UpdatePrometheusPostHook,
		cluster.InstallHelmPostHook,
		cluster.InstallIngressControllerPostHook,
	}
	go cluster.RunPostHooks(postHookFunctions, commonCluster)

	response, err := commonCluster.GetStatus()
	if err != nil {
		log.Errorf("Error during getting cluster status: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, response)
	return
}

// GetClusterStatus retrieves the cluster status
func GetClusterStatus(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": constants.TagGetClusterStatus})

	commonCluster, ok := GetCommonClusterFromRequest(c)
	if ok != true {
		return
	}

	response, err := commonCluster.GetStatus()
	if err != nil {
		log.Errorf("Error during getting status: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during getting status",
			Error:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, response)
	return
}

// GetClusterConfig gets a cluster config
func GetClusterConfig(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": constants.TagFetchClusterConfig})
	commonCluster, ok := GetCommonClusterFromRequest(c)
	if ok != true {
		return
	}
	config, err := commonCluster.GetK8sConfig()
	if err != nil {
		log.Errorf("Error during getting config: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during getting config",
			Error:   err.Error(),
		})
		return
	}

	// Force persist keys
	persistParam := c.DefaultQuery("persist", "false")
	persist, err := strconv.ParseBool(persistParam)
	if err != nil {
		persist = false
	}
	if persist {
		cluster.PersistKubernetesKeys(commonCluster)
	}

	contentType := c.NegotiateFormat(gin.MIMEPlain, gin.MIMEJSON)
	log.Debug("Content-Type: ", contentType)
	switch contentType {
	case gin.MIMEJSON:
		c.JSON(http.StatusOK, components.GetClusterConfigResponse{
			Status: http.StatusOK,
			Data:   string(*config),
		})
	default:
		c.String(http.StatusOK, string(*config))
	}
	return
}

func GetApiEndpoint(c *gin.Context) {
	commonCluster, ok := GetCommonClusterFromRequest(c)
	if !ok {
		return
	}
	endPoint, err := commonCluster.GetAPIEndpoint()
	if err != nil {
		log.Errorf("Error during getting api endpoint: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting endpoint",
			Error:   err.Error(),
		})
		return
	}
	c.String(http.StatusOK, endPoint)
	return
}

// UpdateCluster updates a K8S cluster in the cloud (e.g. autoscale)
func UpdateCluster(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": constants.TagUpdateCluster})

	// bind request body to UpdateClusterRequest struct
	var updateRequest *components.UpdateClusterRequest
	if err := c.BindJSON(&updateRequest); err != nil {
		log.Errorf("Error parsing request: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}
	commonCluster, ok := GetCommonClusterFromRequest(c)
	if ok != true {
		return
	}

	if commonCluster.GetType() != updateRequest.Cloud {
		msg := fmt.Sprintf("Stored cloud type [%s] and request cloud type [%s] not equal", commonCluster.GetType(), updateRequest.Cloud)
		log.Errorf(msg)
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: msg,
		})
		return
	}

	log.Info("Add default values to request if necessarily")

	// set default
	commonCluster.AddDefaultsToUpdate(updateRequest)

	log.Info("Check equality")
	if err := commonCluster.CheckEqualityToUpdate(updateRequest); err != nil {
		log.Errorf("Check changes failed: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Error:   err.Error(),
		})
		return
	}

	if err := updateRequest.Validate(); err != nil {
		log.Errorf("Validation failed: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Error:   err.Error(),
		})
		return
	}

	// TODO check if validation can be applied sooner
	err := commonCluster.UpdateCluster(updateRequest)
	if err != nil {
		// validation failed
		log.Errorf("Update failed: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Error:   err.Error(),
		})
		return
	}
	// save the updated cluster to database
	if err := commonCluster.Persist(); err != nil {
		log.Errorf("Error during cluster save %s", err.Error())
	}

	c.JSON(http.StatusAccepted, components.UpdateClusterResponse{
		Status: http.StatusAccepted,
	})
}

// DeleteCluster deletes a K8S cluster from the cloud
func DeleteCluster(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": constants.TagDeleteCluster})
	commonCluster, ok := GetCommonClusterFromRequest(c)
	if ok != true {
		return
	}
	log.Info("Delete cluster start")

	forceParam := c.DefaultQuery("force", "false")
	force, err := strconv.ParseBool(forceParam)
	if err != nil {
		force = false
	}

	config, err := commonCluster.GetK8sConfig()
	if err != nil && !force {
		log.Errorf("Error during getting kubeconfig: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during getting kubeconfig",
			Error:   err.Error(),
		})
		return
	}

	err = helm.DeleteAllDeployment(config)
	if err != nil {
		log.Errorf("Problem deleting deployment: %s", err)
	}

	err = commonCluster.DeleteCluster()
	if err != nil && !force {
		log.Errorf(errors.Wrap(err, "Error during delete cluster").Error())
		c.JSON(http.StatusInternalServerError, components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error during delete cluster",
			Error:   err.Error(),
		})
		return
	}
	deleteName := commonCluster.GetName()
	deleteId := commonCluster.GetID()
	err = commonCluster.DeleteFromDatabase()
	if err != nil && !force {
		log.Errorf(errors.Wrap(err, "Error during delete cluster from database").Error())
		c.JSON(http.StatusInternalServerError, components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error during delete cluster",
			Error:   err.Error(),
		})
	}

	// Asyncron update prometheus
	go cluster.UpdatePrometheus()

	c.JSON(http.StatusAccepted, components.DeleteClusterResponse{
		Status:     http.StatusAccepted,
		Name:       deleteName,
		Message:    "Cluster deleted successfully",
		ResourceID: deleteId,
	})
	return
}

// FetchClusters fetches all the K8S clusters from the cloud
func FetchClusters(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": constants.TagGetCluster})
	log.Info("Fetching clusters")

	var clusters []model.ClusterModel //TODO change this to CommonClusterStatus
	db := model.GetDB()
	organization := auth.GetCurrentOrganization(c.Request)
	organization.Name = ""
	err := db.Model(organization).Related(&clusters).Error
	if err != nil {
		log.Errorf("Error listing clusters: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error listing clusters",
			Error:   err.Error(),
		})
		return
	}
	response := make([]components.GetClusterStatusResponse, 0)
	for _, cl := range clusters {
		commonCluster, err := cluster.GetCommonClusterFromModel(&cl)
		if err == nil {
			status, err := commonCluster.GetStatus()
			if err != nil {
				//TODO we want skip or return error?
				log.Errorf("get status failed for %s: %s", commonCluster.GetName(), err.Error())
			} else {
				log.Debugf("Append cluster to list: %s", commonCluster.GetName())
				response = append(response, *status)
			}
		} else {
			log.Errorf("convert ClusterModel to CommonCluster failed: %s ", err.Error())
		}
	}
	c.JSON(http.StatusOK, response)
}

// FetchCluster fetch a K8S cluster in the cloud
func FetchCluster(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": constants.TagGetClusterStatus})
	commonCluster, ok := GetCommonClusterFromRequest(c)
	if ok != true {
		return
	}
	log.Info("getting cluster info")
	status, err := commonCluster.GetStatus()
	if err != nil {
		log.Errorf("Error getting cluster: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting cluster",
			Error:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, status)
}

//Status
//func Status(c *gin.Context) {
//	var clusters []cluster.CommonCluster
//	log := logger.WithFields(logrus.Fields{"tag": constants.TagStatus})
//	db := model.GetDB()
//	db.Find(&clusters)
//
//	if len(clusters) == 0 {
//		c.JSON(http.StatusOK, gin.H{"No running clusters found.": http.StatusOK})
//	} else {
//		var clusterStatuses []pods.ClusterStatusResponse
//		for _, cl := range clusters {
//			log.Info("Start listing pods / cluster")
//			var clusterStatusResponse pods.ClusterStatusResponse
//			clusterStatusResponse, err := pods.ListPodsForCluster(&cl)
//			if err == nil {
//				clusterStatuses = append(clusterStatuses, clusterStatusResponse)
//			} else {
//				log.Error(err)
//			}
//
//		}
//		c.JSON(http.StatusOK, gin.H{"clusterStatuses": clusterStatuses})
//	}
//
//}
//
