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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"
	"strings"

	"regexp"

	"github.com/banzaicloud/anchore-image-validator/pkg/apis/security/v1alpha1"
	clientV1alpha1 "github.com/banzaicloud/anchore-image-validator/pkg/clientset/v1alpha1"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/internal/security"
	pkgCommmon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/pkg/security"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

const policyPath string = "policies"

func init() {
	v1alpha1.AddToScheme(scheme.Scheme)
}

func getSecurityClient(c *gin.Context) *clientV1alpha1.SecurityV1Alpha1Client {
	kubeConfig, ok := GetK8sConfig(c)
	if !ok {
		return nil
	}
	config, err := k8sclient.NewClientConfig(kubeConfig)
	if err != nil {
		log.Errorf("Error getting K8s config: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting K8s config",
			Error:   err.Error(),
		})
		return nil
	}

	securityClientSet, err := clientV1alpha1.SecurityConfig(config)
	if err != nil {
		log.Errorf("Error getting SecurityClient: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting SecurityClient",
			Error:   err.Error(),
		})
		return nil
	}
	return securityClientSet
}

// GetScanLog returns image scan results for all deployments
func GetScanLog(c *gin.Context) {
	securityClientSet := getSecurityClient(c)
	if securityClientSet == nil {
		return
	}
	releaseName := c.Param("releaseName")

	audits, err := securityClientSet.Audits(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		err := errors.Wrap(err, "Error during request processing")
		log.Error(err.Error())
		c.JSON(http.StatusInternalServerError, pkgCommmon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error getting scanlogs",
			Error:   err.Error(),
		})
		return
	}

	scanLogList := make([]v1alpha1.AuditSpec, 0)
	for _, audit := range audits.Items {
		scanLog := v1alpha1.AuditSpec{
			ReleaseName: audit.Spec.ReleaseName,
			Resource:    audit.Spec.Resource,
			Action:      audit.Spec.Action,
			Images:      audit.Spec.Images,
			Result:      audit.Spec.Result,
		}
		if len(releaseName) == 0 || audit.Spec.ReleaseName == releaseName {
			scanLogList = append(scanLogList, scanLog)
		}
	}

	c.JSON(http.StatusOK, scanLogList)

}

// GetWhiteLists returns whitelists for all deployments
func GetWhiteLists(c *gin.Context) {
	securityClientSet := getSecurityClient(c)
	if securityClientSet == nil {
		return
	}

	whitelists, err := securityClientSet.Whitelists(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		err := errors.Wrap(err, "Error during request processing")
		log.Error(err.Error())
		c.JSON(http.StatusInternalServerError, pkgCommmon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error getting whitelists",
			Error:   err.Error(),
		})
		return
	}

	releaseWhitelist := make([]security.ReleaseWhiteListItem, 0)
	for _, whitelist := range whitelists.Items {
		whitelistItem := security.ReleaseWhiteListItem{
			Name:   whitelist.Name,
			Owner:  whitelist.Spec.Creator,
			Reason: whitelist.Spec.Reason,
		}
		releaseWhitelist = append(releaseWhitelist, whitelistItem)
	}

	c.JSON(http.StatusOK, releaseWhitelist)

}

// CreateWhiteList creates a whitelist for a deployment
func CreateWhiteList(c *gin.Context) {
	securityClientSet := getSecurityClient(c)
	if securityClientSet == nil {
		return
	}

	var whitelistCreateRequest *security.ReleaseWhiteListItem
	err := c.BindJSON(&whitelistCreateRequest)
	if err != nil {
		err := errors.Wrap(err, "Error parsing request:")
		log.Error(err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during parsing request!",
			Error:   errors.Cause(err).Error(),
		})
		return
	}

	whitelist := v1alpha1.WhiteListItem{
		TypeMeta: metav1.TypeMeta{
			Kind:       "WhiteListItem",
			APIVersion: fmt.Sprintf("%v/%v", v1alpha1.GroupName, v1alpha1.GroupVersion),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: whitelistCreateRequest.Name,
		},
		Spec: v1alpha1.WhiteListSpec{
			Creator: whitelistCreateRequest.Owner,
			Reason:  whitelistCreateRequest.Reason,
		},
	}
	_, err = securityClientSet.Whitelists(metav1.NamespaceDefault).Create(&whitelist)
	if err != nil {
		err := errors.Wrap(err, "Error during request processing")
		log.Error(err.Error())
		c.JSON(http.StatusInternalServerError, pkgCommmon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error creating whitelist",
			Error:   err.Error(),
		})
		return
	}
	c.Status(http.StatusCreated)
}

// DeleteWhiteList deletes a whitelist
func DeleteWhiteList(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "WhiteList name is required!",
			Error:   "WhiteList name is required!",
		})
		return
	}

	securityClientSet := getSecurityClient(c)
	if securityClientSet == nil {
		return
	}

	err := securityClientSet.Whitelists(metav1.NamespaceDefault).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		err := errors.Wrap(err, "Error during request processing")
		log.Error(err.Error())
		c.JSON(http.StatusInternalServerError, pkgCommmon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error deleting whitelist",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, "deleted")
}

func createResponse(c *gin.Context, response http.Response) {
	var responsePayload interface{}
	err := json.NewDecoder(response.Body).Decode(&responsePayload)
	if err != nil {
		log.Error("Error parsing response: %v", err.Error())
		c.JSON(http.StatusInternalServerError, pkgCommmon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error parsing response",
			Error:   err.Error(),
		})
	} else {
		c.JSON(response.StatusCode, responsePayload)
	}
}

// GetPolicies returns image scan results for all deployments
func GetPolicies(c *gin.Context) {

	endPoint := policyPath
	policyId := c.Param("policyId")
	if len(policyId) != 0 {
		endPoint = path.Join(endPoint, policyId)
	}

	commonCluster, ok := getClusterFromRequest(c)
	if !ok {
		return
	}

	if !commonCluster.GetSecurityScan() {
		c.JSON(http.StatusNotFound, pkgCommmon.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "security scan isn't enabled",
			Error:   "security scan isn't enabled",
		})
		return
	}

	anchoreRequest := anchore.AnchoreRequest{
		OrgID:     commonCluster.GetOrganizationId(),
		ClusterID: commonCluster.GetUID(),
		Method:    http.MethodGet,
		URL:       endPoint,
		Body:      nil,
	}
	response, err := anchore.DoAnchoreRequest(anchoreRequest)
	if err != nil {
		log.Error(err)
		c.JSON(http.StatusInternalServerError, pkgCommmon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error",
			Error:   err.Error(),
		})
		return
	}
	defer response.Body.Close()

	createResponse(c, *response)
}

// CreatePolicy returns image scan results for all deployments
func CreatePolicy(c *gin.Context) {

	var policyBundle *security.PolicyBundle
	err := c.BindJSON(&policyBundle)
	if err != nil {
		err := errors.Wrap(err, "Error parsing request:")
		log.Error(err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during parsing request!",
			Error:   errors.Cause(err).Error(),
		})
		return
	}

	commonCluster, ok := getClusterFromRequest(c)
	if !ok {
		return
	}

	if !commonCluster.GetSecurityScan() {
		c.JSON(http.StatusNotFound, pkgCommmon.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "security scan isn't enabled",
			Error:   "security scan isn't enabled",
		})
		return
	}

	anchoreRequest := anchore.AnchoreRequest{
		OrgID:     commonCluster.GetOrganizationId(),
		ClusterID: commonCluster.GetUID(),
		Method:    http.MethodPost,
		URL:       policyPath,
		Body:      policyBundle,
	}
	response, err := anchore.DoAnchoreRequest(anchoreRequest)
	if err != nil {
		log.Error(err)
		c.JSON(http.StatusInternalServerError, pkgCommmon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error",
			Error:   err.Error(),
		})
		return
	}
	defer response.Body.Close()

	createResponse(c, *response)
}

// UpdatePolicies returns image scan results for all deployments
func UpdatePolicies(c *gin.Context) {

	policyId := c.Param("policyId")
	if len(policyId) == 0 {
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "policyId is required!",
			Error:   "policyId is required!",
		})
		return
	}

	commonCluster, ok := getClusterFromRequest(c)
	if !ok {
		return
	}

	if !commonCluster.GetSecurityScan() {
		c.JSON(http.StatusNotFound, pkgCommmon.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "security scan isn't enabled",
			Error:   "security scan isn't enabled",
		})
		return
	}

	anchoreRequest := anchore.AnchoreRequest{
		OrgID:     commonCluster.GetOrganizationId(),
		ClusterID: commonCluster.GetUID(),
		Method:    http.MethodPut,
		URL:       path.Join(policyPath, policyId),
		Body:      nil,
	}

	if active, _ := strconv.ParseBool(c.Query("active")); active {
		anchoreRequest.Method = http.MethodGet
		response, err := anchore.DoAnchoreRequest(anchoreRequest)
		if err != nil {
			log.Error(err)
			c.JSON(http.StatusInternalServerError, pkgCommmon.ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "Error",
				Error:   err.Error(),
			})
			return
		}

		respBody, _ := ioutil.ReadAll(response.Body)
		policyBundle := []security.PolicyBundleRecord{}
		json.Unmarshal(respBody, &policyBundle)
		policyBundle[0].Active = true
		anchoreRequest.Method = http.MethodPut
		anchoreRequest.Body = policyBundle[0]
	} else {
		var policyBundle *security.PolicyBundleRecord
		err := c.BindJSON(&policyBundle)
		if err != nil {
			err := errors.Wrap(err, "Error parsing request:")
			log.Error(err.Error())
			c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: "Error during parsing request!",
				Error:   errors.Cause(err).Error(),
			})
			return
		}
		anchoreRequest.Body = policyBundle
	}

	response, err := anchore.DoAnchoreRequest(anchoreRequest)
	if err != nil {
		log.Error(err)
		c.JSON(http.StatusInternalServerError, pkgCommmon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error",
			Error:   err.Error(),
		})
		return
	}
	defer response.Body.Close()

	createResponse(c, *response)
}

// DeletePolicy returns image scan results for all deployments
func DeletePolicy(c *gin.Context) {

	policyId := c.Param("policyId")
	if len(policyId) == 0 {
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "policyId is required!",
			Error:   "policyId is required!",
		})
		return
	}

	commonCluster, ok := getClusterFromRequest(c)
	if !ok {
		return
	}

	if !commonCluster.GetSecurityScan() {
		c.JSON(http.StatusNotFound, pkgCommmon.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "security scan isn't enabled",
			Error:   "security scan isn't enabled",
		})
		return
	}

	anchoreRequest := anchore.AnchoreRequest{
		OrgID:     commonCluster.GetOrganizationId(),
		ClusterID: commonCluster.GetUID(),
		Method:    http.MethodDelete,
		URL:       path.Join(policyPath, policyId),
		Body:      nil,
	}
	response, err := anchore.DoAnchoreRequest(anchoreRequest)
	if err != nil {
		log.Error(err)
		c.JSON(http.StatusInternalServerError, pkgCommmon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error",
			Error:   err.Error(),
		})
		return
	}
	defer response.Body.Close()

	createResponse(c, *response)

}

// GetImageDeployments list deployments by image
func GetImageDeployments(c *gin.Context) {
	imageDigest := c.Param("imageDigest")
	releaseMap := make(map[string]bool)

	re := regexp.MustCompile("^sha256:[a-f0-9]{64}$")
	if !re.MatchString(imageDigest) {
		err := fmt.Errorf("Invalid imageID format: %s", imageDigest)
		log.Error(err)
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting K8s config",
			Error:   err.Error(),
		})
		return
	}

	kubeConfig, ok := GetK8sConfig(c)
	if !ok {
		return
	}

	// Get active helm deployments
	log.Info("Get deployments")
	activeReleases, err := helm.ListDeployments(nil, c.Query("tag"), kubeConfig)
	if err != nil {
		log.Error("Error listing deployments: ", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error listing deployments",
			Error:   err.Error(),
		})
		return
	}

	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		log.Errorf("Error getting K8s config: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting K8s config",
			Error:   err.Error(),
		})
		return
	}
	// Get all pods from cluster
	pods, err := listPods(client, "", "")
	if err != nil {
		log.Errorf("Error getting pods from cluster: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting pods from cluster",
			Error:   err.Error(),
		})
		return
	}
	// Example status
	//	- containerID: docker://a8130dc313a40b0eb9151685ba41f84cd0e4bb7e2888c52691590ff8a22a2e6b
	//	image: banzaicloud/pipeline:0.4.0-dev29
	//	imageID: docker-pullable://banzaicloud/pipeline@sha256:5042ef1a5415dae8330583448584be2bb592053416b7db5fc41389a717cc52ab
	for _, p := range pods {
		for _, status := range p.Status.ContainerStatuses {
			if getImageDigest(status.ImageID) == imageDigest {
				releaseMap[p.Labels["release"]] = true
			}
		}
		for _, status := range p.Status.InitContainerStatuses {
			if getImageDigest(status.ImageID) == imageDigest {
				releaseMap[p.Labels["release"]] = true
			}
		}
	}
	releases := ListHelmReleases(c, activeReleases, releaseMap)

	c.JSON(http.StatusOK, releases)
}

func getImageDigest(imageID string) string {

	image := strings.Split(imageID, "@")
	if len(image) > 1 {
		return image[1]
	}
	return ""
}

// GetWhitelistSet will return a WhitelistSet
func GetWhitelistSet(c *gin.Context) (map[string]bool, bool) {
	securityClientSet := getSecurityClient(c)
	releaseWhitelist := make(map[string]bool)
	if securityClientSet == nil {
		return releaseWhitelist, false
	}
	whitelists, err := securityClientSet.Whitelists(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		log.Warnf("can not fetch WhiteList: %s", err.Error())
		return releaseWhitelist, false
	}
	for _, whitelist := range whitelists.Items {
		releaseWhitelist[whitelist.ObjectMeta.Name] = true
	}
	log.Debugf("Whitelist set: %#v", releaseWhitelist)
	return releaseWhitelist, true
}

// GetReleaseScanLog will return a ReleaseScanlog
func GetReleaseScanLog(c *gin.Context) (map[string]bool, bool) {
	securityClientSet := getSecurityClient(c)
	releaseScanLogReject := make(map[string]bool)
	if securityClientSet == nil {
		return releaseScanLogReject, false
	}
	audits, err := securityClientSet.Audits(metav1.NamespaceAll).List(metav1.ListOptions{LabelSelector: "fakerelease=false"})
	if err != nil {
		log.Warnf("can not fetch ScanLog: %s", err.Error())
		return releaseScanLogReject, false
	}
	for _, audit := range audits.Items {
		if audit.Spec.Action == "reject" {
			releaseScanLogReject[audit.Spec.ReleaseName] = true
		}
	}
	log.Debugf("ReleaseScanLogReject set: %#v", releaseScanLogReject)
	return releaseScanLogReject, true
}

// SecurytiScanEnabled checks if security scan is enabled in pipeline
func SecurytiScanEnabled(c *gin.Context) {

	if viper.GetBool("anchore.enabled") {
		c.JSON(http.StatusOK, gin.H{
			"enabled": true,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"enabled": false,
	})
	return
}
