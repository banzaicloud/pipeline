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

	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
)

// Models copied from generated client package.
// TODO: import these from a generated server model package
type InstallSecretRequest struct {
	Namespace string                                  `json:"namespace"`
	Spec      map[string]InstallSecretRequestSpecItem `json:"spec,omitempty"`
}

type InstallSecretRequestSpecItem struct {
	Source    string            `json:"source,omitempty"`
	SourceMap map[string]string `json:"sourceMap,omitempty"`
}

type InstallSecretResponse struct {
	Name     string `json:"name"`
	Sourcing string `json:"sourcing"`
}

// InstallSecretToCluster installs a particular secret to a cluster's namespace.
func InstallSecretToCluster(c *gin.Context) {
	commonCluster, ok := getClusterFromRequest(c)
	if !ok {
		return
	}

	logger := correlationid.Logger(log, c)

	var request InstallSecretRequest
	if err := c.BindJSON(&request); err != nil {
		logger.WithError(err).Debug("failed to parse request")

		c.AbortWithStatusJSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}

	secretRequest := cluster.InstallSecretRequest{
		Namespace: request.Namespace,
		Spec:      map[string]cluster.InstallSecretRequestSpecItem{},
	}

	for key, spec := range request.Spec {
		secretRequest.Spec[key] = cluster.InstallSecretRequestSpecItem{
			Source:    spec.Source,
			SourceMap: spec.SourceMap,
		}
	}

	secretName := c.Param("secretName")

	secretSource, err := cluster.InstallSecret(commonCluster, secretName, secretRequest)

	if err != nil {
		errorHandler.Handle(emperror.With(
			emperror.Wrap(err, "failed to install secret into cluster"),
			"cluster-id", commonCluster.GetID(),
			"organization-id", commonCluster.GetOrganizationId(),
			"secret", secretName,
		))

		c.AbortWithStatusJSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error installing secret into cluster",
			Error:   err.Error(),
		})
		return
	}

	response := InstallSecretResponse{
		Name:     secretSource.Name,
		Sourcing: string(secretSource.Sourcing),
	}

	c.JSON(http.StatusOK, response)
}
