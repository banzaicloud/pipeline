// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"net/http"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster/supported"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/gin-gonic/gin"
)

// GetSupportedClusterList sends back the supported cluster list
func GetSupportedClusterList(c *gin.Context) {

	log.Info("Start getting supported clusters")

	c.JSON(http.StatusOK, pkgCluster.SupportedClustersResponse{
		Items: []pkgCluster.SupportedClusterItem{
			{
				Name:    "Amazon EKS",
				Key:     "unknown",
				Enabled: false,
				Icon:    "assets/images/aws-eks.png",
			},
			{
				Name:    "Azure Kubernetes Service",
				Key:     pkgCluster.Azure,
				Enabled: true,
				Icon:    "assets/images/azure.png",
			},
			{
				Name:    "Google Kubernetes Engine",
				Key:     pkgCluster.Google,
				Enabled: true,
				Icon:    "assets/images/google.png",
			},
			{
				Name:    "Kubernetes Cluster",
				Key:     pkgCluster.Kubernetes,
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

	request := &pkgCluster.CloudInfoRequest{
		OrganizationId: organizationID,
		SecretId:       secretId,
		Filter: &pkgCluster.CloudInfoFilter{
			Fields: filterFields,
			InstanceType: &pkgCluster.InstanceFilter{
				Location: location,
			},
			KubernetesFilter: &pkgCluster.KubernetesFilter{
				Location: location,
			},
			ImageFilter: &pkgCluster.ImageFilter{
				Location: location,
				Tags:     tags,
			},
		},
	}

	if resp, err := processCloudInfo(cloudType, request); err != nil {
		log.Errorf("Error during getting cloud info: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
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
func processCloudInfo(cloudType string, r *pkgCluster.CloudInfoRequest) (*pkgCluster.GetCloudInfoResponse, error) {
	log.Info("Create cloud info model")
	m, err := supported.GetCloudInfoModel(cloudType, r)
	if err != nil {
		return nil, err
	}

	log.Info("Process filtering")
	return supported.ProcessFilter(m, r)
}
