package api

import (
	"fmt"
	"net/http"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/gin-gonic/gin"
)

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
	commonCluster, ok := getClusterFromRequest(c)
	if ok != true {
		return
	}

	if commonCluster.GetCloud() != updateRequest.Cloud {
		msg := fmt.Sprintf("Stored cloud type [%s] and request cloud type [%s] not equal", commonCluster.GetCloud(), updateRequest.Cloud)
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
