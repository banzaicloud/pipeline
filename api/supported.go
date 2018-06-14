package api

import (
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster/supported"
	"github.com/gin-gonic/gin"
	"net/http"
)

// GetSupportedClusterList sends back the supported cluster list
func GetSupportedClusterList(c *gin.Context) {

	log.Info("Start getting supported clusters")

	organizationID := auth.GetCurrentOrganization(c.Request).IDString()
	log.Infof("Organization id: %s", organizationID)

	c.JSON(http.StatusOK, components.SupportedClustersResponse{
		Items: []components.SupportedClusterItem{
			{
				Name:    "Amazon EC2",
				Key:     constants.Amazon,
				Enabled: true,
				Icon:    "assets/images/amazon.png",
			},
			{
				Name:    "Amazon EKS",
				Key:     "unknown",
				Enabled: false,
				Icon:    "assets/images/aws-eks.png",
			},
			{
				Name:    "Azure Kubernetes Service",
				Key:     constants.Azure,
				Enabled: true,
				Icon:    "assets/images/azure.png",
			},
			{
				Name:    "Google Kubernetes Engine",
				Key:     constants.Google,
				Enabled: true,
				Icon:    "assets/images/google.png",
			},
			{
				Name:    "Kubernetes Cluster",
				Key:     constants.Kubernetes,
				Enabled: true,
				Icon:    "assets/images/kubernetes.png",
			},
			{
				Name:    "Red Hat OpenShift",
				Key:     "unknown",
				Enabled: false,
				Icon:    "assets/images/open_shift.png",
			},
			{
				Name:    "Oracle Kubernetes Engine",
				Key:     "unknown",
				Enabled: false,
				Icon:    "assets/images/oracle.png",
			},
			{
				Name:    "Alibaba Kubernetes Container Service",
				Key:     "unknown",
				Enabled: false,
				Icon:    "assets/images/alibaba_cloud.png",
			},
			{
				Name:    "DigitalOcean Kubernetes",
				Key:     "unknown",
				Enabled: false,
				Icon:    "assets/images/digital_ocean.png",
			},
		},
	})

}

// GetCloudInfo sends back the supported locations/k8sVersions/machineTypes
func GetCloudInfo(c *gin.Context) {

	log.Info("Start getting cloud info")

	organizationID := auth.GetCurrentOrganization(c.Request).ID
	log.Infof("Organization id: %d", organizationID)

	cloudType := c.Param("cloudtype")
	log.Debugf("Cloud type: %s", cloudType)

	filterFields := getFieldsFromQuery(c)
	log.Debugf("Filter fields: %v", filterFields)

	tags := getTagsFromQuery(c)
	log.Debugf("Tags: %v", tags)

	secretId := getSecretIdFromQuery(c)
	log.Debugf("Secret id: %s", secretId)

	location := getLocationFromQuery(c)
	log.Debugf("Location: %s", location)

	request := &components.CloudInfoRequest{
		OrganizationId: organizationID,
		SecretId:       secretId,
		Filter: &components.CloudInfoFilter{
			Fields: filterFields,
			InstanceType: &components.InstanceFilter{
				Location: location,
			},
			KubernetesFilter: &components.KubernetesFilter{
				Location: location,
			},
			ImageFilter: &components.ImageFilter{
				Location: location,
				Tags:     tags,
			},
		},
	}

	if resp, err := processCloudInfo(cloudType, request); err != nil {
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

// getFieldsFromQuery returns fields from query
func getFieldsFromQuery(c *gin.Context) []string {
	return c.QueryArray("fields")
}

// getTagsFromQuery returns tags from query
func getTagsFromQuery(c *gin.Context) (tags []*string) {
	array := c.QueryArray("tags")

	for _, a := range array {
		tags = append(tags, &a)
	}

	return
}

// getSecretIdFromQuery returns secret id from query
func getSecretIdFromQuery(c *gin.Context) string {
	return c.Query("secret_id")
}

// getLocationFromQuery returns location from query
func getLocationFromQuery(c *gin.Context) string {
	return c.Query("location")
}

// processCloudInfo returns the cloud info with the supported fields
func processCloudInfo(cloudType string, r *components.CloudInfoRequest) (*components.GetCloudInfoResponse, error) {
	log.Info("Create cloud info model")
	m, err := supported.GetCloudInfoModel(cloudType, r)
	if err != nil {
		return nil, err
	}

	log.Info("Process filtering")
	return supported.ProcessFilter(m, r)
}
