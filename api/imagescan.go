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

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	apiclient "github.com/banzaicloud/pipeline/client"
	anchore "github.com/banzaicloud/pipeline/internal/security"
	"github.com/banzaicloud/pipeline/pkg/common"
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
		c.JSON(httpStatusCode, common.ErrorResponse{
			Code:    httpStatusCode,
			Message: "Error",
			Error:   "Missing imageDigest",
		})
		return
	}
	doAnchoreGetRequest(c, endPoint)
}

// ScanImages scans images
func ScanImages(c *gin.Context) {

	var images []apiclient.ClusterImage
	endPoint := imagscanEndPoint
	err := c.BindJSON(&images)
	if err != nil {
		err := errors.Wrap(err, "Error parsing request:")
		log.Error(err.Error())
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
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
		common.ErrorResponseWithStatus(c, http.StatusNotFound, errors.New(anchore.SecurityScanNotEnabledMessage))
		return
	}

	var anchorePost anchoreImagePostBody
	for i := range images {
		anchorePost.Tag = images[i].ImageName + ":" + images[i].ImageTag
		anchorePost.Digest = ""
		anchoreRequest := anchore.AnchoreRequest{
			OrgID:     commonCluster.GetOrganizationId(),
			ClusterID: commonCluster.GetUID(),
			Method:    http.MethodPost,
			URL:       endPoint,
			Body:      anchorePost,
		}
		response, err := anchore.DoAnchoreRequest(anchoreRequest)
		if err != nil {
			log.Error(err)
			httpStatusCode := http.StatusInternalServerError
			c.JSON(httpStatusCode, common.ErrorResponse{
				Code:    httpStatusCode,
				Message: "Error",
				Error:   err.Error(),
			})
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
		c.JSON(httpStatusCode, common.ErrorResponse{
			Code:    httpStatusCode,
			Message: "Error",
			Error:   "Missing imageDigest",
		})
		return
	}
	endPoint = path.Join(endPoint, "/vuln/all")
	doAnchoreGetRequest(c, endPoint)
}

func doAnchoreGetRequest(c *gin.Context, endPoint string) {
	commonCluster, ok := getClusterFromRequest(c)
	if !ok {
		return
	}

	if !commonCluster.GetSecurityScan() {
		common.ErrorResponseWithStatus(c, http.StatusNotFound, errors.New(anchore.SecurityScanNotEnabledMessage))
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
		httpStatusCode := http.StatusInternalServerError
		c.JSON(httpStatusCode, common.ErrorResponse{
			Code:    httpStatusCode,
			Message: "Error",
			Error:   err.Error(),
		})
		return
	}
	defer response.Body.Close()
	createResponse(c, *response)
}
