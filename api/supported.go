package api

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/banzai-types/components"
	"net/http"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster/supported"
)

// GetSupportedClusterList sends back the supported cluster list
func GetSupportedClusterList(c *gin.Context) {

	log := logger.WithFields(logrus.Fields{"tag": "GetSupportedClusterList"})

	log.Info("Start getting supported clusters")

	organizationID := auth.GetCurrentOrganization(c.Request).IDString()
	log.Infof("Organization id: %s", organizationID)

	c.JSON(http.StatusOK, components.SupportedClustersResponse{
		Items: []components.SupportedClusterItem{
			{
				Name: "Amazon Web Services",
				Key:  constants.Amazon,
			},
			{
				Name: "Azure Container Service",
				Key:  constants.Azure,
			},
			{
				Name: "Google Kubernetes Engine",
				Key:  constants.Google,
			},
			{
				Name: "Build Your Own Cluster",
				Key:  constants.BYOC,
			},
			{
				Name: "Dummy cluster",
				Key:  constants.Dummy,
			},
		},
	})

}

// GetSupportedFilters sends back the supported filter words
func GetSupportedFilters(c *gin.Context) {

	log := logger.WithFields(logrus.Fields{"tag": "GetSupportedFilters"})
	log.Info("Start getting filter keys")

	c.JSON(http.StatusOK, components.SupportedFilters{
		Keys: supported.Keywords,
	})

}

// GetCloudInfo sends back the supported locations/k8sVersions/machineTypes
func GetCloudInfo(c *gin.Context) {

	log := logger.WithFields(logrus.Fields{"tag": "GetCloudInfo"})
	log.Info("Start getting cloud info")

	organizationID := auth.GetCurrentOrganization(c.Request).ID
	log.Infof("Organization id: %d", organizationID)

	cloudType := c.Param("cloudtype")
	log.Debugf("Cloud type: %s", cloudType)

	log.Info("Binding request")
	var request components.CloudInfoRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Errorf("Error during binding request: %s", err.Error())
	}

	log.Info("Binding request succeeded")
	request.OrganizationId = organizationID
	if resp, err := processCloudInfo(cloudType, &request); err != nil {
		log.Errorf("Error during getting cloud info: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during getting cloud info",
			Error:   err.Error(),
		})
	} else {
		log.Debugf("Cloud info: %#v", resp)
		c.JSON(http.StatusOK, resp)
	}
}

// processCloudInfo returns the cloud info with the supported fields
func processCloudInfo(cloudType string, r *components.CloudInfoRequest) (*components.GetCloudInfoResponse, error) {
	log.Info("Create cloud info model")
	if m, err := supported.GetCloudInfoModel(cloudType, r); err != nil {
		return nil, err
	} else {
		log.Info("Process filtering")
		return supported.ProcessFilter(m, r)
	}
}
