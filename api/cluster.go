package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/database"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/model/defaults"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	resourceHelper "k8s.io/kubernetes/pkg/api/v1/resource"
)

var log *logrus.Logger

const (
	awsLabelMaster = "node-role.kubernetes.io/master"

	statusReady    = "Ready"
	statusNotReady = "Not ready"
	statusUnknown  = "Unknown"
	readyTrue      = "True"
	readyFalse     = "False"
)

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
	userID := auth.GetCurrentUser(c.Request).ID

	posthookFunctions := createClusterRequest.PostHookFunctions
	log.Infof("Get posthook function(s) by name(s): %v", posthookFunctions)
	var ph []cluster.PostFunctioner
	for _, f := range posthookFunctions {
		ph = append(ph, cluster.HookMap[f])
	}

	log.Infof("Found posthooks: %v", ph)

	commonCluster, err := CreateCluster(&createClusterRequest, orgID, userID, ph)
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
func CreateCluster(createClusterRequest *pkgCluster.CreateClusterRequest, organizationID, userID uint,
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
	database := database.GetDB()
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
	commonCluster, err := cluster.CreateCommonClusterFromRequest(createClusterRequest, organizationID, userID)
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

		sshSecretId, err := commonCluster.GetModel().AddSshKey()
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

	// Apply PostHooks
	// These are hardcoded posthooks maybe we will want a bit more dynamic
	postHookFunctions := cluster.BasePostHookFunctions

	if postHooks != nil && len(postHooks) != 0 {
		postHookFunctions = append(postHookFunctions, postHooks...)
	}

	err = cluster.RunPostHooks(postHookFunctions, commonCluster)

	if err != nil {
		log.Errorf("Error during running cluster posthooks: %s", err.Error())
		return err
	}

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

	userId := auth.GetCurrentUser(c.Request).ID

	go postUpdateCluster(commonCluster, updateRequest, userId)

	c.JSON(http.StatusAccepted, pkgCluster.UpdateClusterResponse{
		Status: http.StatusAccepted,
	})
}

// postUpdateCluster updates a cluster (ASYNC)
func postUpdateCluster(commonCluster cluster.CommonCluster, updateRequest *pkgCluster.UpdateClusterRequest, userId uint) error {

	err := commonCluster.UpdateCluster(updateRequest, userId)
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

	log.Info("deploy autoscaler")
	if err := cluster.DeployClusterAutoscaler(commonCluster); err != nil {
		log.Errorf("Error during update cluster status: %s", err.Error())
		return err
	}

	log.Info("Add labels to nodes")

	return cluster.LabelNodes(commonCluster)
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
	db := database.GetDB()
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

// ClusterHEAD checks the cluster ready
func ClusterHEAD(c *gin.Context) {

	commonCluster, ok := GetCommonClusterFromRequest(c)
	if ok != true {
		return
	}

	log.Info("getting cluster")
	_, err := commonCluster.GetClusterDetails()
	if err != nil {
		log.Errorf("Error getting cluster: %s", err.Error())
		c.Status(http.StatusBadRequest)
		return
	}

	c.Status(http.StatusOK)

}

// GetClusterDetails fetch a K8S cluster in the cloud
func GetClusterDetails(c *gin.Context) {
	commonCluster, ok := GetCommonClusterFromRequest(c)
	if ok != true {
		return
	}
	log.Info("getting cluster details")
	details, err := commonCluster.GetClusterDetails()
	if err != nil {
		log.Errorf("Error getting cluster: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting cluster",
			Error:   err.Error(),
		})
		return
	}

	log.Info("Start getting API endpoint")
	endpoint, err := commonCluster.GetAPIEndpoint()
	if err != nil {
		log.Warnf("Error during getting API endpoint: %s", err.Error())
	}
	details.Endpoint = endpoint

	log.Info("Add resource summary to node(s)")
	if err := addResourceSummaryToDetails(commonCluster, details); err != nil {
		log.Warnf("Error during adding summary: %s", err.Error())
	}

	secret, err := commonCluster.GetSecretWithValidation()
	if err != nil {
		log.Errorf("Error getting cluster secret: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error getting cluster secret",
			Error:   err.Error(),
		})
		return
	}

	details.SecretId = secret.ID
	details.SecretName = secret.Name

	c.JSON(http.StatusOK, details)
}

// addResourceSummaryToDetails adds resource summary to all node in each pool
func addResourceSummaryToDetails(commonCluster cluster.CommonCluster, details *pkgCluster.DetailsResponse) error {

	log.Info("get K8S config")
	kubeConfig, err := commonCluster.GetK8sConfig()
	if err != nil {
		return err
	}

	log.Info("get k8S connection")
	client, err := helm.GetK8sConnection(kubeConfig)
	if err != nil {
		return err
	}

	// add node summary
	log.Info("Add summary to nodes")
	for name := range details.NodePools {

		if err := addNodeSummaryToDetails(client, details, name); err != nil {
			return err
		}

	}

	// add master summary, in case of Amazon
	if commonCluster.GetType() == pkgCluster.Amazon {

		log.Info("cloud type is amazon, add master summary")
		if err := addMasterSummaryToDetails(client, details); err != nil {
			return err
		}

	}

	// add total summary
	log.Info("add total summary")
	return addTotalSummaryToDetails(client, details)
}

// addTotalSummaryToDetails calculate all resource summary
func addTotalSummaryToDetails(client *kubernetes.Clientset, details *pkgCluster.DetailsResponse) (err error) {

	log.Info("list nodes")
	var nodeList *v1.NodeList
	nodeList, err = client.CoreV1().Nodes().List(meta_v1.ListOptions{})
	if err != nil {
		return
	}

	log.Infof("nodes [%d]", len(nodeList.Items))

	log.Info("list pods")
	var pods []v1.Pod
	pods, err = listPods(client, "")
	if err != nil {
		return
	}

	log.Infof("pods [%d]", len(pods))

	log.Info("Calculate total requests/limits/capacity/allocatable")
	requests, limits := calculatePodsTotalRequestsAndLimits(pods)
	capacity, allocatable := calculateNodesTotalCapacityAndAllocatable(nodeList.Items)

	resourceSummary := getResourceSummary(capacity, allocatable, requests, limits)
	details.TotalSummary = resourceSummary

	return
}

// addMasterSummaryToDetails add master resource summary in case of Amazon
func addMasterSummaryToDetails(client *kubernetes.Clientset, details *pkgCluster.DetailsResponse) error {

	selector := fmt.Sprintf("%s=", awsLabelMaster)

	log.Info("List nodes with selector: %s", selector)
	nodes, err := client.CoreV1().Nodes().List(meta_v1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return err
	}

	log.Infof("nodes [%d]", len(nodes.Items))

	if len(nodes.Items) != 0 {

		log.Info("add master resource summary")

		master := nodes.Items[0]
		resourceSummary, err := getResourceSummaryFromNode(client, &master)
		if err != nil {
			return err
		}

		details.Master = make(map[string]pkgCluster.ResourceSummary)
		details.Master[master.Name] = *resourceSummary

		log.Info("master summary added")
	}

	return nil

}

// addNodeSummaryToDetails adds node resource summary
func addNodeSummaryToDetails(client *kubernetes.Clientset, details *pkgCluster.DetailsResponse, nodePoolName string) error {

	selector := fmt.Sprintf("%s=%s", pkgCommon.LabelKey, nodePoolName)

	log.Infof("List nodes with selector: %s", selector)

	nodes, err := client.CoreV1().Nodes().List(meta_v1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return err
	}

	log.Infof("nodes [%d]", len(nodes.Items))

	details.NodePools[nodePoolName].ResourceSummary = make(map[string]pkgCluster.ResourceSummary)

	for _, node := range nodes.Items {

		log.Infof("add summary to node [%s] in nodepool [s]", node.Name, nodePoolName)

		resourceSummary, err := getResourceSummaryFromNode(client, &node)
		if err != nil {
			return err
		}
		details.NodePools[nodePoolName].ResourceSummary[node.Name] = *resourceSummary
		log.Infof("summary added to node [%s] in nodepool [%s]", node.Name, nodePoolName)
	}

	return nil
}

// getResourceSummaryFromNode return resource summary for the given node
func getResourceSummaryFromNode(client *kubernetes.Clientset, node *v1.Node) (*pkgCluster.ResourceSummary, error) {

	fieldSelector, err := fields.ParseSelector("spec.nodeName=" + node.Name)
	if err != nil {
		return nil, err
	}

	log.Infof("start getting requests and limits of all pods in all namespace")
	requests, limits, err := getAllPodsRequestsAndLimitsInAllNamespace(client, fieldSelector.String())
	if err != nil {
		return nil, err
	}

	var capCPU resource.Quantity
	var capMemory resource.Quantity
	var allocCPU resource.Quantity
	var allocMemory resource.Quantity
	if cpu := node.Status.Capacity.Cpu(); cpu != nil {
		capCPU = *cpu
	}

	if mem := node.Status.Capacity.Memory(); mem != nil {
		capMemory = *mem
	}

	if cpu := node.Status.Allocatable.Cpu(); cpu != nil {
		allocCPU = *cpu
	}

	if mem := node.Status.Allocatable.Memory(); mem != nil {
		allocMemory = *mem
	}

	// set capacity map
	capacity := map[v1.ResourceName]resource.Quantity{
		v1.ResourceCPU:    capCPU,
		v1.ResourceMemory: capMemory,
	}

	// set allocatable map
	allocatable := map[v1.ResourceName]resource.Quantity{
		v1.ResourceCPU:    allocCPU,
		v1.ResourceMemory: allocMemory,
	}

	resourceSummary := getResourceSummary(capacity, allocatable, requests, limits)
	resourceSummary.Status = getNodeStatus(node)

	return resourceSummary, nil

}

// getNodeStatus returns the node actual status
func getNodeStatus(node *v1.Node) string {

	for _, condition := range node.Status.Conditions {
		if condition.Type == statusReady {
			switch condition.Status {
			case readyTrue:
				return statusReady
			case readyFalse:
				return statusNotReady
			default:
				return statusUnknown

			}
		}
	}

	return ""

}

// getResourceSummary returns ResourceSummary type with the given data
func getResourceSummary(capacity, allocatable, requests, limits map[v1.ResourceName]resource.Quantity) *pkgCluster.ResourceSummary {

	var capMem string
	var capCPU string
	var allMem string
	var allCPU string
	var reqMem string
	var reqCPU string
	var limitMem string
	var limitCPU string

	if cpu, ok := capacity[v1.ResourceCPU]; ok {
		capCPU = cpu.String()
	}

	if memory, ok := capacity[v1.ResourceMemory]; ok {
		capMem = memory.String()
	}

	if cpu, ok := allocatable[v1.ResourceCPU]; ok {
		allCPU = cpu.String()
	}

	if memory, ok := allocatable[v1.ResourceMemory]; ok {
		allMem = memory.String()
	}

	if value, ok := requests[v1.ResourceCPU]; ok {
		reqCPU = value.String()
	}

	if value, ok := requests[v1.ResourceMemory]; ok {
		reqMem = value.String()
	}

	if value, ok := limits[v1.ResourceCPU]; ok {
		limitCPU = value.String()
	}

	if value, ok := limits[v1.ResourceMemory]; ok {
		limitMem = value.String()
	}

	return &pkgCluster.ResourceSummary{
		Cpu: &pkgCluster.CPU{
			ResourceSummaryItem: pkgCluster.ResourceSummaryItem{
				Capacity:    capCPU,
				Allocatable: allCPU,
				Limit:       limitCPU,
				Request:     reqCPU,
			},
		},
		Memory: &pkgCluster.Memory{
			ResourceSummaryItem: pkgCluster.ResourceSummaryItem{
				Capacity:    capMem,
				Allocatable: allMem,
				Limit:       limitMem,
				Request:     reqMem,
			},
		},
	}
}

func getAllPodsRequestsAndLimitsInAllNamespace(client *kubernetes.Clientset, fieldSelector string) (map[v1.ResourceName]resource.Quantity, map[v1.ResourceName]resource.Quantity, error) {

	log.Infof("list pods with field selector: %s", fieldSelector)
	podList, err := listPods(client, fieldSelector)
	if err != nil {
		return nil, nil, err
	}

	log.Infof("pods [%d]", len(podList))
	log.Infof("calculate requests and limits")
	req, limits := calculatePodsTotalRequestsAndLimits(podList)
	return req, limits, nil
}

// listPods returns list of pods in all namspace
func listPods(client *kubernetes.Clientset, fieldSelector string) (pods []v1.Pod, err error) {

	log.Info("List namespaces")
	var namespaces *v1.NamespaceList
	namespaces, err = client.CoreV1().Namespaces().List(meta_v1.ListOptions{})
	if err != nil {
		return
	}

	log.Infof("namespaces: %v", namespaces.Items)

	var podList *v1.PodList
	for _, np := range namespaces.Items {

		log.Infof("List pods in namespace [%s] with selector: %s", np.Name, fieldSelector)

		podList, err = client.CoreV1().Pods(np.Name).List(meta_v1.ListOptions{
			FieldSelector: fieldSelector,
		})
		if err != nil {
			return
		}

		pods = append(pods, podList.Items...)
	}

	log.Debugf(" pod list [%d]", len(pods))

	return
}

// calculateNodesTotalCapacityAndAllocatable calculates capacity and allocatable of the given nodes
func calculateNodesTotalCapacityAndAllocatable(nodeList []v1.Node) (caps map[v1.ResourceName]resource.Quantity, allocs map[v1.ResourceName]resource.Quantity) {

	caps, allocs = map[v1.ResourceName]resource.Quantity{}, map[v1.ResourceName]resource.Quantity{}
	for _, node := range nodeList {

		nodeCaps, nodeAllocs := nodeCapacityAndAllocatable(&node)
		for nodeCapName, nodeCapValue := range nodeCaps {
			if value, ok := caps[nodeCapName]; !ok {
				caps[nodeCapName] = *nodeCapValue.Copy()
			} else {
				value.Add(nodeCapValue)
				caps[nodeCapName] = value
			}
		}

		for nodeAllocName, nodeAllocValue := range nodeAllocs {
			if value, ok := allocs[nodeAllocName]; !ok {
				allocs[nodeAllocName] = *nodeAllocValue.Copy()
			} else {
				value.Add(nodeAllocValue)
				allocs[nodeAllocName] = value
			}
		}
	}

	return
}

// nodeCapacityAndAllocatable returns the given node's capacity and allocatable
func nodeCapacityAndAllocatable(node *v1.Node) (caps map[v1.ResourceName]resource.Quantity, allocs map[v1.ResourceName]resource.Quantity) {
	caps, allocs = make(map[v1.ResourceName]resource.Quantity), make(map[v1.ResourceName]resource.Quantity)

	nodeCap := node.Status.Capacity
	nodeAlloc := node.Status.Allocatable

	if nodeCap.Memory() != nil {
		caps[v1.ResourceMemory] = *nodeCap.Memory()
	}

	if nodeCap.Cpu() != nil {
		caps[v1.ResourceCPU] = *nodeCap.Cpu()
	}

	if nodeAlloc.Memory() != nil {
		allocs[v1.ResourceMemory] = *nodeAlloc.Memory()
	}

	if nodeAlloc.Cpu() != nil {
		allocs[v1.ResourceCPU] = *nodeAlloc.Cpu()
	}

	return
}

// calculatePodsTotalRequestsAndLimits calculates requests and limits of all the given pods
func calculatePodsTotalRequestsAndLimits(podList []v1.Pod) (reqs map[v1.ResourceName]resource.Quantity, limits map[v1.ResourceName]resource.Quantity) {
	reqs, limits = map[v1.ResourceName]resource.Quantity{}, map[v1.ResourceName]resource.Quantity{}
	for _, pod := range podList {
		podReqs, podLimits := resourceHelper.PodRequestsAndLimits(&pod)
		for podReqName, podReqValue := range podReqs {
			if value, ok := reqs[podReqName]; !ok {
				reqs[podReqName] = *podReqValue.Copy()
			} else {
				value.Add(podReqValue)
				reqs[podReqName] = value
			}
		}
		for podLimitName, podLimitValue := range podLimits {
			if value, ok := limits[podLimitName]; !ok {
				limits[podLimitName] = *podLimitValue.Copy()
			} else {
				value.Add(podLimitValue)
				limits[podLimitName] = value
			}
		}
	}
	return
}

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
