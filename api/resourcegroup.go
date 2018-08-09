package api

import (
	"net/http"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// GetResourceGroups lists resource groups by secret
func GetResourceGroups(c *gin.Context) {

	orgID := auth.GetCurrentOrganization(c.Request).ID
	secretId := c.Query("secretId")

	log := log.WithFields(logrus.Fields{"secret": secretId, "org": orgID})

	log.Info("Start listing resource groups")

	groups, err := cluster.ListResourceGroups(orgID, secretId)
	if err != nil {
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during listing resource groups",
			Error:   err.Error(),
		})
		return
	}

	log.Infof("resource groups found: %v", groups)

	c.JSON(http.StatusOK, groups)

}

// AddResourceGroups creates a new resource group
func AddResourceGroups(c *gin.Context) {

	orgID := auth.GetCurrentOrganization(c.Request).ID
	log := log.WithFields(logrus.Fields{"org": orgID})

	log.Info("Start adding resource group")

	log.Debug("Bind json into CreateClusterRequest struct")
	var request CreateResourceGroupRequest
	if err := c.BindJSON(&request); err != nil {
		log.Errorf("error during parsing request: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during parsing request",
			Error:   err.Error(),
		})
		return
	}

	if err := cluster.CreateOrUpdateResourceGroup(orgID, request.SecretId, request.Name, request.Location); err != nil {
		log.Errorf("error during creating resource group: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "error during creating resource groups",
			Error:   err.Error(),
		})
		return
	}

	log.Infof("resource group [%s] created successfully", request.Name)

	c.JSON(http.StatusCreated, CreateResourceGroupResponse{
		Name: request.Name,
	})

}

// DeleteResourceGroups deletes resource group by name
func DeleteResourceGroups(c *gin.Context) {

	orgID := auth.GetCurrentOrganization(c.Request).ID
	secretId := c.Query("secretId")
	name := c.Param("name")

	log := log.WithFields(logrus.Fields{"secret": secretId, "org": orgID, "bucketName": name})

	log.Info("Start deleting resource group")

	if err := cluster.DeleteResourceGroup(orgID, secretId, name); err != nil {
		log.Errorf("error during deleting resource group: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "error during deleting resource group",
			Error:   err.Error(),
		})
		return
	}

	log.Info("resource group deleted successfully")

	c.Status(http.StatusAccepted)

}
