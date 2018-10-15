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
	"path"

	apiclient "github.com/banzaicloud/pipeline/client"
	"github.com/banzaicloud/pipeline/internal/security"
	pkgCommmon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type anchoreImagePostBody struct {
	Tag    string `json:"tag,omitempty"`
	Digest string `json:"digest,omitempty"`
}

const imagscanEndPoint = "images"

// GetScanResult list scan result
func GetScanResult(c *gin.Context) {

	endPoint := imagscanEndPoint
	imageDigest := c.Param("imagedigest")
	if len(imageDigest) != 0 {
		endPoint = path.Join(endPoint, imageDigest)
	} else {
		log.Error("Missing imageDigest")
		httpStatusCode := http.StatusNotFound
		c.JSON(httpStatusCode, pkgCommmon.ErrorResponse{
			Code:    httpStatusCode,
			Message: "Error",
			Error:   "Missing imageDigest",
		})
	}

	commonCluster, ok := getClusterFromRequest(c)
	if !ok {
		return
	}
	response, err := anchore.MakeAnchoreRequest(commonCluster.GetOrganizationId(), commonCluster.GetUID(), http.MethodGet, endPoint, nil)
	if err != nil {
		internalServerError(c, err)
		return
	}
	defer response.Body.Close()
	createResponse(c, *response)
}

// ScanImages scans images
func ScanImages(c *gin.Context) {

	var images []apiclient.ClusterImage
	endPoint := imagscanEndPoint
	err := c.BindJSON(&images)
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
	var anchorePost anchoreImagePostBody
	for i := range images {
		anchorePost.Tag = images[i].ImageName + ":" + images[i].ImageTag
		anchorePost.Digest = ""
		// if imageDigest set, anchore will intiate force image scanning
		// anchorePost.Digest = images[i].ImageDigest
		// if anchorePost.Digest != "" {
		// 	endPoint = "images?force=true"
		// }
		response, err := anchore.MakeAnchoreRequest(commonCluster.GetOrganizationId(), commonCluster.GetUID(), http.MethodPost, endPoint, anchorePost)
		if err != nil {
			internalServerError(c, err)
			return
		}
		defer response.Body.Close()
		createResponse(c, *response)
	}
}

// GetImageVulnerabilities list image vulnerabilities
func GetImageVulnerabilities(c *gin.Context) {

	endPoint := imagscanEndPoint
	imageDigest := c.Param("imagedigest")
	if len(imageDigest) != 0 {
		endPoint = path.Join(endPoint, imageDigest)
	} else {
		log.Error("Missing imageDigest")
		httpStatusCode := http.StatusNotFound
		c.JSON(httpStatusCode, pkgCommmon.ErrorResponse{
			Code:    httpStatusCode,
			Message: "Error",
			Error:   "Missing imageDigest",
		})
	}
	endPoint = path.Join(endPoint, "/vuln/all")
	commonCluster, ok := getClusterFromRequest(c)
	if !ok {
		return
	}
	response, err := anchore.MakeAnchoreRequest(commonCluster.GetOrganizationId(), commonCluster.GetUID(), http.MethodGet, endPoint, nil)
	if err != nil {
		internalServerError(c, err)
		return
	}
	defer response.Body.Close()
	createResponse(c, *response)
}

func internalServerError(c *gin.Context, err error) {
	log.Error(err)
	httpStatusCode := http.StatusInternalServerError
	c.JSON(httpStatusCode, pkgCommmon.ErrorResponse{
		Code:    httpStatusCode,
		Message: "Error",
		Error:   err.Error(),
	})
}
