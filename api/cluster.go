package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/model/defaults"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	pipelineSsh "github.com/banzaicloud/pipeline/ssh"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

// Simple init for logging
func init() {
	log = config.Logger()
}

//ParseField is to restrict other query TODO investigate to just pass the hasmap
func ParseField(c *gin.Context) map[string]interface{} {
	value := c.Param("id")
	field := c.DefaultQuery("field", "id")
	return map[string]interface{}{field: value}
}

// UpdateMonitoring updating prometheus
func UpdateMonitoring(c *gin.Context) {
	cluster.UpdatePrometheus()
	c.String(http.StatusOK, "OK")
	return
}

// GetCommonClusterFromFilter get filtered cluster
func GetCommonClusterFromFilter(c *gin.Context, filter map[string]interface{}) (cluster.CommonCluster, bool) {
	modelCluster, err := model.QueryCluster(filter)
	if err != nil {
		log.Errorf("Cluster not found: %s", err.Error())
		c.JSON(http.StatusNotFound, pkgCommon.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "Cluster not found",
			Error:   err.Error(),
		})
		return nil, false
	}

	if len(modelCluster) == 0 {
		log.Error("Empty cluster list")
		c.JSON(http.StatusNotFound, pkgCommon.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "Cluster not found",
			Error:   "",
		})
		return nil, false
	}

	commonCLuster, err := cluster.GetCommonClusterFromModel(&modelCluster[0])
	if err != nil {
		log.Errorf("GetCommonClusterFromModel failed: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return nil, false
	}
	return commonCLuster, true
}

// GetCommonClusterFromRequest just a simple getter to build commonCluster object this handles error messages directly
func GetCommonClusterFromRequest(c *gin.Context) (cluster.CommonCluster, bool) {
	filter := ParseField(c)

	// Filter for organisation
	filter["organization_id"] = c.Request.Context().Value(auth.CurrentOrganization).(*auth.Organization).ID
	return GetCommonClusterFromFilter(c, filter)
}

//GetCommonClusterNameFromRequest get cluster name from cluster request
func GetCommonClusterNameFromRequest(c *gin.Context) (string, bool) {
	commonCluster, ok := GetCommonClusterFromRequest(c)
	if ok != true {
		return "", false
	}
	clusterName := commonCluster.GetName()
	log.Debugln("clusterName:", clusterName)
	return clusterName, true
}

//CreateClusterRequest gin handler
func CreateClusterRequest(c *gin.Context) {
	//TODO refactor logging here

	log.Info("Cluster creation started")

	log.Debug("Bind json into CreateClusterRequest struct")
	// bind request body to struct
	var createClusterRequest pkgCluster.CreateClusterRequest
	if err := c.BindJSON(&createClusterRequest); err != nil {
		log.Error(errors.Wrap(err, "Error parsing request"))
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}

	orgID := auth.GetCurrentOrganization(c.Request).ID

	posthookFunctions := createClusterRequest.PostHookFunctions
	log.Infof("Get posthook function(s) by name(s): %v", posthookFunctions)
	var ph []cluster.PostFunctioner
	for _, f := range posthookFunctions {
		ph = append(ph, cluster.HookMap[f])
	}

	log.Infof("Found posthooks: %v", ph)

	commonCluster, err := CreateCluster(&createClusterRequest, orgID, ph)
	if err != nil {
		c.JSON(err.Code, err)
		return
	}

	c.JSON(http.StatusAccepted, pkgCluster.CreateClusterResponse{
		Name:       commonCluster.GetName(),
		ResourceID: commonCluster.GetID(),
	})
}

// CreateCluster creates a K8S cluster in the cloud
func CreateCluster(createClusterRequest *pkgCluster.CreateClusterRequest, organizationID uint,
	postHooks []cluster.PostFunctioner) (cluster.CommonCluster, *pkgCommon.ErrorResponse) {

	if len(createClusterRequest.ProfileName) != 0 {
		log.Infof("Fill data from profile[%s]", createClusterRequest.ProfileName)
		profile, err := defaults.GetProfile(createClusterRequest.Cloud, createClusterRequest.ProfileName)
		if err != nil {
			return nil, &pkgCommon.ErrorResponse{
				Code:    http.StatusNotFound,
				Message: "Error during getting profile",
				Error:   err.Error(),
			}
		}

		log.Info("Create profile response")
		profileResponse := profile.GetProfile()

		log.Info("Create clusterRequest from profile")
		newRequest, err := profileResponse.CreateClusterRequest(createClusterRequest)
		if err != nil {
			return nil, &pkgCommon.ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: "Error creating request from profile",
				Error:   err.Error(),
			}
		}

		createClusterRequest = newRequest

		log.Infof("Modified clusterRequest: %v", createClusterRequest)

	}

	log.Debug("Parsing request succeeded")

	log.Info("Searching entry with name: ", createClusterRequest.Name)

	// check exists cluster name
	var existingCluster model.ClusterModel
	database := model.GetDB()
	database.First(&existingCluster, map[string]interface{}{"name": createClusterRequest.Name, "organization_id": organizationID})

	if existingCluster.ID != 0 {
		// duplicated entry
		err := fmt.Errorf("duplicate entry: %s", existingCluster.Name)
		return nil, &pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Error:   err.Error(),
		}
	}

	log.Info("Creating new entry with cloud type: ", createClusterRequest.Cloud)

	// TODO check validation
	// This is the common part of cluster flow
	commonCluster, err := cluster.CreateCommonClusterFromRequest(createClusterRequest, organizationID)
	if err != nil {
		return nil, &pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Error:   err.Error(),
		}
	}

	log.Infof("Validate secret[%s]", createClusterRequest.SecretId)
	if _, err := commonCluster.GetSecretWithValidation(); err != nil {
		return nil, &pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during getting secret",
			Error:   err.Error(),
		}
	}
	log.Info("Secret validation passed")

	// Persist the cluster in Database
	err = commonCluster.Persist(pkgCluster.Creating, pkgCluster.CreatingMessage)
	if err != nil {
		return nil, &pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
			Error:   err.Error(),
		}
	}

	log.Info("Validate creation fields")
	if err := commonCluster.ValidateCreationFields(createClusterRequest); err != nil {
		commonCluster.UpdateStatus(pkgCluster.Error, err.Error())
		return nil, &pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Error:   err.Error(),
		}
	}

	log.Info("Validation passed")

	go postCreateCluster(commonCluster, postHooks)
	return commonCluster, nil
}

// postCreateCluster creates a cluster (ASYNC)
func postCreateCluster(commonCluster cluster.CommonCluster, postHooks []cluster.PostFunctioner) error {

	// Check if public ssh key is needed for the cluster. If so and there is generate one and store it Vault
	if len(commonCluster.GetSshSecretId()) == 0 && commonCluster.RequiresSshPublicKey() {
		log.Infof("Generating Ssh Key for the cluster")

		sshSecretId, err := pipelineSsh.KeyAdd(commonCluster.GetOrganizationId(), commonCluster.GetID())
		if err != nil {
			log.Errorf("Generating Ssh Key for organization id=%s, cluster id=%s failed: %s", commonCluster.GetOrganizationId(), commonCluster.GetID(), err.Error())
			return err
		}

		if err := commonCluster.SaveSshSecretId(sshSecretId); err != nil {
			log.Errorf("Error during cluster creation: %s", err.Error())
			return err
		}
	}

	// Create cluster
	err := commonCluster.CreateCluster()
	if err != nil {
		log.Errorf("Error during cluster creation: %s", err.Error())
		commonCluster.UpdateStatus(pkgCluster.Error, err.Error())
		return err
	}

	err = commonCluster.UpdateStatus(pkgCluster.Running, pkgCluster.RunningMessage)
	if err != nil {
		log.Errorf("Error during updating cluster status: %s", err.Error())
		return err
	}

	// Apply PostHooks
	// These are hardcoded posthooks maybe we will want a bit more dynamic
	postHookFunctions := cluster.BasePostHookFunctions

	if postHooks != nil && len(postHooks) != 0 {
		postHookFunctions = append(postHookFunctions, postHooks...)
	}

	cluster.RunPostHooks(postHookFunctions, commonCluster)

	return nil
}

// GetClusterStatus retrieves the cluster status
func GetClusterStatus(c *gin.Context) {

	commonCluster, ok := GetCommonClusterFromRequest(c)
	if ok != true {
		return
	}

	response, err := commonCluster.GetStatus()
	if err != nil {
		log.Errorf("Error during getting status: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
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
	commonCluster, ok := GetCommonClusterFromRequest(c)
	if ok != true {
		return
	}
	config, err := commonCluster.GetK8sConfig()
	if err != nil {
		log.Errorf("Error during getting config: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
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
		c.JSON(http.StatusOK, pkgCluster.GetClusterConfigResponse{
			Status: http.StatusOK,
			Data:   string(config),
		})
	default:
		c.String(http.StatusOK, string(config))
	}
	return
}

// GetApiEndpoint returns the Kubernetes Api endpoint
func GetApiEndpoint(c *gin.Context) {

	log.Info("Start getting API endpoint")

	log.Info("Create common cluster model from request")
	commonCluster, ok := GetCommonClusterFromRequest(c)
	if !ok {
		return
	}

	log.Info("Start getting API endpoint")
	endPoint, err := commonCluster.GetAPIEndpoint()
	if err != nil {
		log.Errorf("Error during getting api endpoint: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting endpoint",
			Error:   err.Error(),
		})
		return
	}

	log.Debugf("API endpoint: %s", endPoint)

	c.String(http.StatusOK, endPoint)
	return
}

// UpdateCluster updates a K8S cluster in the cloud (e.g. autoscale)
func UpdateCluster(c *gin.Context) {

	// bind request body to UpdateClusterRequest struct
	var updateRequest *pkgCluster.UpdateClusterRequest
	if err := c.BindJSON(&updateRequest); err != nil {
		log.Errorf("Error parsing request: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
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
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: msg,
		})
		return
	}

	log.Info("Check cluster status")
	status, err := commonCluster.GetStatus()
	if err != nil {
		log.Errorf("Error checking status: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error checking status",
			Error:   err.Error(),
		})
		return
	}

	log.Infof("Cluster status: %s", status.Status)

	if status.Status != pkgCluster.Running {
		err := fmt.Errorf("cluster is not in %s state yet", pkgCluster.Running)
		log.Errorf("Error during checking cluster status: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during checking cluster status",
			Error:   err.Error(),
		})
		return
	}

	log.Info("Add default values to request if necessarily")

	// set default
	commonCluster.AddDefaultsToUpdate(updateRequest)

	log.Info("Check equality")
	if err := commonCluster.CheckEqualityToUpdate(updateRequest); err != nil {
		log.Errorf("Check changes failed: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Error:   err.Error(),
		})
		return
	}

	if err := updateRequest.Validate(); err != nil {
		log.Errorf("Validation failed: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Error:   err.Error(),
		})
		return
	}

	// save the updated cluster to database
	if err := commonCluster.Persist(pkgCluster.Updating, pkgCluster.UpdatingMessage); err != nil {
		log.Errorf("Error during cluster save %s", err.Error())
	}

	go postUpdateCluster(commonCluster, updateRequest)

	c.JSON(http.StatusAccepted, pkgCluster.UpdateClusterResponse{
		Status: http.StatusAccepted,
	})
}

// postUpdateCluster updates a cluster (ASYNC)
func postUpdateCluster(commonCluster cluster.CommonCluster, updateRequest *pkgCluster.UpdateClusterRequest) error {

	err := commonCluster.UpdateCluster(updateRequest)
	if err != nil {
		// validation failed
		log.Errorf("Update failed: %s", err.Error())
		commonCluster.UpdateStatus(pkgCluster.Error, err.Error())
		return err
	}

	err = commonCluster.UpdateStatus(pkgCluster.Running, pkgCluster.RunningMessage)
	if err != nil {
		log.Errorf("Error during update cluster status: %s", err.Error())
		return err
	}

	return cluster.DeployClusterAutoscaler(commonCluster)
}

// DeleteCluster deletes a K8S cluster from the cloud
func DeleteCluster(c *gin.Context) {
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

	go postDeleteCluster(commonCluster, force)

	deleteName := commonCluster.GetName()
	deleteId := commonCluster.GetID()

	c.JSON(http.StatusAccepted, pkgCluster.DeleteClusterResponse{
		Status:     http.StatusAccepted,
		Name:       deleteName,
		ResourceID: deleteId,
	})
}

// postDeleteCluster deletes a cluster (ASYNC)
func postDeleteCluster(commonCluster cluster.CommonCluster, force bool) error {

	err := commonCluster.UpdateStatus(pkgCluster.Deleting, pkgCluster.DeletingMessage)
	if err != nil {
		log.Errorf("Error during updating cluster status: %s", err.Error())
		return err
	}

	// get kubeconfig
	c, err := commonCluster.GetK8sConfig()
	if err != nil && !force {
		log.Errorf("Error during getting kubeconfig: %s", err.Error())
		commonCluster.UpdateStatus(pkgCluster.Error, err.Error())
		return err
	}

	// delete deployments
	err = helm.DeleteAllDeployment(c)
	if err != nil {
		log.Errorf("Problem deleting deployment: %s", err)
	}

	// delete cluster
	err = commonCluster.DeleteCluster()
	if err != nil && !force {
		log.Errorf(errors.Wrap(err, "Error during delete cluster").Error())
		commonCluster.UpdateStatus(pkgCluster.Error, err.Error())
		return err
	}

	// delete cluster from database
	deleteName := commonCluster.GetName()
	err = commonCluster.DeleteFromDatabase()
	if err != nil && !force {
		log.Errorf(errors.Wrap(err, "Error during delete cluster from database").Error())
		commonCluster.UpdateStatus(pkgCluster.Error, err.Error())
		return err
	}

	// Asyncron update prometheus
	go cluster.UpdatePrometheus()

	// clean statestore
	log.Info("Clean cluster's statestore folder ")
	if err := cluster.CleanStateStore(deleteName); err != nil {
		log.Errorf("Statestore cleaning failed: %s", err.Error())
	} else {
		log.Info("Cluster's statestore folder cleaned")
	}
	return nil
}

// FetchClusters fetches all the K8S clusters from the cloud
func FetchClusters(c *gin.Context) {
	log.Info("Fetching clusters")

	var clusters []model.ClusterModel //TODO change this to CommonClusterStatus
	db := model.GetDB()
	organization := auth.GetCurrentOrganization(c.Request)
	organization.Name = ""
	err := db.Model(organization).Related(&clusters).Error
	if err != nil {
		log.Errorf("Error listing clusters: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error listing clusters",
			Error:   err.Error(),
		})
		return
	}
	response := make([]pkgCluster.GetClusterStatusResponse, 0)
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

// ReRunPostHooks handles {cluster_id}/posthooks API request
func ReRunPostHooks(c *gin.Context) {

	log.Info("Get common cluster")
	commonCluster, ok := GetCommonClusterFromRequest(c)
	if ok != true {
		return
	}

	var ph cluster.RunPostHook
	if err := c.BindJSON(&ph); err != nil {
		log.Errorf("error during binding request: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "error during binding request",
			Error:   err.Error(),
		})
		return
	}

	var posthooks []cluster.PostFunctioner
	if len(ph.Functions) == 0 {
		posthooks = cluster.BasePostHookFunctions
	} else {
		for _, f := range ph.Functions {
			posthooks = append(posthooks, cluster.HookMap[f])
		}
	}

	log.Infof("Cluster id: %d", commonCluster.GetID())
	log.Infof("Run posthook(s): %v", posthooks)

	go cluster.RunPostHooks(posthooks, commonCluster)

	c.Status(http.StatusOK)
}

// FetchCluster fetch a K8S cluster in the cloud
func FetchCluster(c *gin.Context) {
	commonCluster, ok := GetCommonClusterFromRequest(c)
	if ok != true {
		return
	}
	log.Info("getting cluster info")
	status, err := commonCluster.GetClusterDetails()
	if err != nil {
		log.Errorf("Error getting cluster: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
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

// InstallSecretsToCluster add all secrets from a repo to a cluster's namespace combined into one global secret named as the repo
func InstallSecretsToCluster(c *gin.Context) {
	commonCluster, ok := GetCommonClusterFromRequest(c)
	if !ok {
		return
	}

	// bind request body to UpdateClusterRequest struct
	var request pkgSecret.InstallSecretsToClusterRequest
	if err := c.BindJSON(&request); err != nil {
		log.Errorf("Error parsing request: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}

	secretSources, err := cluster.InstallSecrets(commonCluster, &request.Query, request.Namespace)

	if err != nil {
		log.Errorf("Error installing secrets [%v] into cluster [%d]: %s", request.Query, commonCluster.GetID(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error installing secrets into cluster",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, secretSources)
}
