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
	"net/http"
	"path"
	"strings"

	"regexp"

	"time"

	"github.com/banzaicloud/anchore-image-validator/pkg/apis/security/v1alpha1"
	clientV1alpha1 "github.com/banzaicloud/anchore-image-validator/pkg/clientset/v1alpha1"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/internal/security"
	pkgCommmon "github.com/banzaicloud/pipeline/pkg/common"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/pkg/security"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

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

	audits, err := securityClientSet.Audits(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		err := errors.Wrap(err, "Error during request processing")
		log.Error(err.Error())
		httpStatusCode := http.StatusInternalServerError
		c.JSON(httpStatusCode, pkgCommmon.ErrorResponse{
			Code:    httpStatusCode,
			Message: "Error getting scanlogs",
			Error:   err.Error(),
		})
		return
	}

	scanLogList := make([]security.ScanLogItem, 0)
	for _, audit := range audits.Items {
		scanLog := security.ScanLogItem{
			ReleaseName: audit.Spec.ReleaseName,
			Resource:    audit.Spec.Resource,
			Action:      audit.Spec.Action,
			Image:       audit.Spec.Image,
			Result:      audit.Spec.Result,
		}
		scanLogList = append(scanLogList, scanLog)
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
		httpStatusCode := http.StatusInternalServerError
		c.JSON(httpStatusCode, pkgCommmon.ErrorResponse{
			Code:    httpStatusCode,
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
		httpStatusCode := http.StatusInternalServerError
		c.JSON(httpStatusCode, pkgCommmon.ErrorResponse{
			Code:    httpStatusCode,
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
		httpStatusCode := http.StatusBadRequest
		c.JSON(httpStatusCode, pkgCommmon.ErrorResponse{
			Code:    httpStatusCode,
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
		httpStatusCode := http.StatusInternalServerError
		c.JSON(httpStatusCode, pkgCommmon.ErrorResponse{
			Code:    httpStatusCode,
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
		httpStatusCode := http.StatusInternalServerError
		c.JSON(httpStatusCode, pkgCommmon.ErrorResponse{
			Code:    httpStatusCode,
			Message: "Error parsing response",
			Error:   err.Error(),
		})
	} else {
		c.JSON(response.StatusCode, responsePayload)
	}
}

// GetPolicies returns image scan results for all deployments
func GetPolicies(c *gin.Context) {

	endPoint := "policies"
	policyId := c.Param("policyId")
	if len(policyId) != 0 {
		endPoint = path.Join(endPoint, policyId)
	}

	commonCluster, ok := getClusterFromRequest(c)
	if !ok {
		return
	}
	response, err := anchore.MakeAnchoreRequest(commonCluster.GetOrganizationId(), commonCluster.GetUID(), http.MethodGet, endPoint, nil)
	if err != nil {
		log.Error(err)
		httpStatusCode := http.StatusInternalServerError
		c.JSON(httpStatusCode, pkgCommmon.ErrorResponse{
			Code:    httpStatusCode,
			Message: "Error",
			Error:   err.Error(),
		})
		return
	}
	defer response.Body.Close()

	createResponse(c, *response)
}

// CreatePolicies returns image scan results for all deployments
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
	response, err := anchore.MakeAnchoreRequest(commonCluster.GetOrganizationId(), commonCluster.GetUID(), http.MethodPost, "policies", policyBundle)
	if err != nil {
		log.Error(err)
		httpStatusCode := http.StatusInternalServerError
		c.JSON(httpStatusCode, pkgCommmon.ErrorResponse{
			Code:    httpStatusCode,
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
		httpStatusCode := http.StatusBadRequest
		c.JSON(httpStatusCode, pkgCommmon.ErrorResponse{
			Code:    httpStatusCode,
			Message: "policyId is required!",
			Error:   "policyId is required!",
		})
		return
	}

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

	commonCluster, ok := getClusterFromRequest(c)
	if !ok {
		return
	}
	response, err := anchore.MakeAnchoreRequest(commonCluster.GetOrganizationId(), commonCluster.GetUID(), http.MethodPut, path.Join("policies", policyId), policyBundle)
	if err != nil {
		log.Error(err)
		httpStatusCode := http.StatusInternalServerError
		c.JSON(httpStatusCode, pkgCommmon.ErrorResponse{
			Code:    httpStatusCode,
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
		httpStatusCode := http.StatusBadRequest
		c.JSON(httpStatusCode, pkgCommmon.ErrorResponse{
			Code:    httpStatusCode,
			Message: "policyId is required!",
			Error:   "policyId is required!",
		})
		return
	}

	commonCluster, ok := getClusterFromRequest(c)
	if !ok {
		return
	}
	response, err := anchore.MakeAnchoreRequest(commonCluster.GetOrganizationId(), commonCluster.GetUID(), http.MethodDelete, path.Join("policies", policyId), nil)
	if err != nil {
		log.Error(err)
		httpStatusCode := http.StatusInternalServerError
		c.JSON(httpStatusCode, pkgCommmon.ErrorResponse{
			Code:    httpStatusCode,
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

	// Get WhiteList set
	releaseWhitelist, ok := GetWhitelistSet(c)
	if !ok {
		return
	}
	log.Debugf("Whitelist set: %#v", releaseWhitelist)

	kubeConfig, ok := GetK8sConfig(c)
	if !ok {
		log.Warnf("whitelist data is not valid: %#v", releaseWhitelist)
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

	releases := []pkgHelm.ListDeploymentResponse{}
	if activeReleases != nil && len(activeReleases.Releases) > 0 {
		for _, r := range activeReleases.Releases {
			if ok := releaseMap[r.Name]; ok {
				createdAt := time.Unix(r.Info.FirstDeployed.Seconds, 0)
				updated := time.Unix(r.Info.LastDeployed.Seconds, 0)
				chartName := r.GetChart().GetMetadata().GetName()

				body := pkgHelm.ListDeploymentResponse{
					Name:         r.Name,
					Chart:        helm.GetVersionedChartName(r.Chart.Metadata.Name, r.Chart.Metadata.Version),
					ChartName:    chartName,
					ChartVersion: r.GetChart().GetMetadata().GetVersion(),
					Version:      r.Version,
					UpdatedAt:    updated,
					Status:       r.Info.Status.Code.String(),
					Namespace:    r.Namespace,
					CreatedAt:    createdAt,
				}
				//Add WhiteListed flag if present
				if _, ok := releaseWhitelist[r.Name]; ok {
					body.WhiteListed = ok
				}
				releases = append(releases, body)
			}
		}
	} else {
		log.Info("There are no installed charts.")
	}
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
		releaseWhitelist[whitelist.Spec.ReleaseName] = true
	}
	log.Debugf("Whitelist set: %#v", releaseWhitelist)
	return releaseWhitelist, true
}
