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

	"github.com/gin-gonic/gin"

	intlHelm "github.com/banzaicloud/pipeline/internal/helm"
	pkgCommmon "github.com/banzaicloud/pipeline/pkg/common"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/src/helm"
)

// GetK8sConfig returns the Kubernetes config
func GetK8sConfig(c *gin.Context) ([]byte, bool) {
	commonCluster, ok := getClusterFromRequest(c)
	if ok != true {
		return nil, false
	}
	kubeConfig, err := commonCluster.GetK8sConfig()
	if err != nil {
		log.Errorf("Error getting config: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting kubeconfig",
			Error:   err.Error(),
		})
		return nil, false
	}
	return kubeConfig, true
}

// ListHelmReleases list helm releases
func ListHelmReleases(c *gin.Context, releases []intlHelm.Release, releaseMap map[string]bool) []pkgHelm.ListDeploymentResponse {
	// Get WhiteList set
	releaseWhitelist, ok := GetWhitelistSet(c)
	if !ok {
		log.Warnf("whitelist data is not valid: %#v", releaseWhitelist)
	}
	releaseScanLogReject, ok := GetReleaseScanLog(c)
	if !ok {
		log.Warnf("scanlog data is not valid: %#v", releaseScanLogReject)
	}

	releasesResponse := make([]pkgHelm.ListDeploymentResponse, 0)
	if releases != nil && len(releases) > 0 {
		for _, release := range releases {
			createdAt := release.ReleaseInfo.FirstDeployed
			updated := release.ReleaseInfo.LastDeployed
			chartName := release.ChartName

			body := pkgHelm.ListDeploymentResponse{
				Name:         release.ReleaseName,
				Chart:        helm.GetVersionedChartName(release.ChartName, release.Version),
				ChartName:    chartName,
				ChartVersion: release.Version,
				Version:      release.ReleaseVersion,
				UpdatedAt:    updated,
				Status:       release.ReleaseInfo.Status,
				Namespace:    release.Namespace,
				CreatedAt:    createdAt,
			}

			// Add WhiteListed flag if present
			if _, ok := releaseWhitelist[release.ReleaseName]; ok {
				body.WhiteListed = ok
			}
			if _, ok := releaseScanLogReject[release.ReleaseName]; ok {
				body.Rejected = ok
			}
			if _, ok := releaseMap[release.ReleaseName]; ok {
				releasesResponse = append(releasesResponse, body)
			}
		}
	} else {
		log.Info("There are no installed charts.")
	}
	return releasesResponse
}
