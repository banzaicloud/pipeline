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
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	apiclient "github.com/banzaicloud/pipeline/client"
	pkgCommmon "github.com/banzaicloud/pipeline/pkg/common"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

// ListImages list all used images in cluster
func ListImages(c *gin.Context) {
	kubeConfig, ok := GetK8sConfig(c)
	if !ok {
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

	imageList, err := listAllImages(client, "")
	if err != nil {
		err := errors.Wrap(err, "Error during request processing")
		log.Error(err.Error())
		httpStatusCode := http.StatusInternalServerError
		c.JSON(httpStatusCode, pkgCommmon.ErrorResponse{
			Code:    httpStatusCode,
			Message: "Error getting Pods",
			Error:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, imageList)
}

// GetDeploymentImages list all used images in deployment
func GetDeploymentImages(c *gin.Context) {
	release := c.Param("name")
	log.Infof("getting images for deployment: [%s]", release)

	kubeConfig, ok := GetK8sConfig(c)
	if !ok {
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

	selector := fmt.Sprintf("%s=%s", pkgHelm.HelmReleaseNameLabel, release)
	log.Infof("Label selector: %s", selector)

	imageList, err := listAllImages(client, selector)
	if err != nil {
		err := errors.Wrap(err, "error during request processing")
		log.Error(err.Error())

		httpStatusCode := http.StatusInternalServerError
		c.JSON(httpStatusCode, pkgCommmon.ErrorResponse{
			Code:    httpStatusCode,
			Message: "Error getting Pods",
			Error:   err.Error(),
		})
		return
	}

	if len(imageList) == 0 {
		selector = fmt.Sprintf("%s=%s", pkgHelm.HelmReleaseNameLabelLegacy, release)
		log.Infof("Label selector: %s", selector)

		imageList, err = listAllImages(client, selector)

		if err != nil {
			err := errors.Wrap(err, "error during request processing")
			log.Error(err.Error())

			httpStatusCode := http.StatusInternalServerError
			c.JSON(httpStatusCode, pkgCommmon.ErrorResponse{
				Code:    httpStatusCode,
				Message: "Error getting Pods",
				Error:   err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, imageList)
}

func listAllImages(client *kubernetes.Clientset, labelSelector string) ([]*apiclient.ClusterImage, error) {
	var err error
	var podList []v1.Pod
	podList, err = listPods(client, "", labelSelector)
	if err != nil {
		return nil, err
	}

	imageList := make([]*apiclient.ClusterImage, 0)
	for _, pod := range podList {
		images := getPodImages(pod)
		imageList = append(imageList, images...)
	}
	deDupList := removeDuplicatedImages(imageList)
	return deDupList, nil
}

func getPodImages(pod v1.Pod) []*apiclient.ClusterImage {

	images := make([]*apiclient.ClusterImage, 0)
	for _, container := range pod.Status.ContainerStatuses {
		fullName := strings.Split(container.Image, ":")
		var name string
		tag := "latest"
		if len(fullName) > 1 {
			name = fullName[0]
			tag = fullName[1]
		}
		fullDigest := strings.Split(container.ImageID, "@")
		var digest string
		if len(fullDigest) > 1 {
			digest = fullDigest[1]
		} else {
			continue
		}

		image := apiclient.ClusterImage{
			ImageName:   name,
			ImageTag:    tag,
			ImageDigest: digest,
		}
		images = append(images, &image)
	}
	return images
}

func removeDuplicatedImages(images []*apiclient.ClusterImage) []*apiclient.ClusterImage {
	found := make(map[string]bool)
	j := 0
	for i, image := range images {
		if image.ImageDigest != "" {
			if !found[image.ImageDigest] {
				found[image.ImageDigest] = true
				images[j] = images[i]
				j++
			}
		}
	}
	return images[:j]
}
