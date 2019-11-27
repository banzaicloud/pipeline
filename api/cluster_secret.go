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

	"emperror.dev/emperror"
	"github.com/gin-gonic/gin"

	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
)

// Models copied from generated client package.
// TODO: import these from a generated server model package
type InstallSecretRequest struct {
	SourceSecretName string                                  `json:"sourceSecretName,omitempty"`
	Namespace        string                                  `json:"namespace"`
	Spec             map[string]InstallSecretRequestSpecItem `json:"spec,omitempty"`
}

type InstallSecretRequestSpecItem struct {
	Source    string            `json:"source,omitempty"`
	SourceMap map[string]string `json:"sourceMap,omitempty"`
	Value     string            `json:"value,omitempty"`
}

type InstallSecretResponse struct {
	Name string `json:"name"`
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
		SourceSecretName: request.SourceSecretName,
		Namespace:        request.Namespace,
		Spec:             map[string]cluster.InstallSecretRequestSpecItem{},
	}

	for key, spec := range request.Spec {
		secretRequest.Spec[key] = cluster.InstallSecretRequestSpecItem{
			Source:    spec.Source,
			SourceMap: spec.SourceMap,
			Value:     spec.Value,
		}
	}

	secretName := c.Param("secretName")

	// Either spec is not defined (empty) or at least one spec is not a value or empty
	needsSecret := len(secretRequest.Spec) == 0
	for _, spec := range secretRequest.Spec {
		if spec.Source != "" || len(spec.SourceMap) != 0 || spec.Value == "" {
			needsSecret = true
			break
		}
	}

	// If there is no separate pipeline secret name use the same as the cluster request name
	if needsSecret && secretRequest.SourceSecretName == "" {
		secretRequest.SourceSecretName = secretName
	}

	err := cluster.InstallSecret(commonCluster, secretName, secretRequest)

	if err == cluster.ErrSecretNotFound {
		ginutils.ReplyWithErrorResponse(c, &pkgCommon.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "secret not found",
		})

		return
	} else if err == cluster.ErrKubernetesSecretAlreadyExists {
		ginutils.ReplyWithErrorResponse(c, &pkgCommon.ErrorResponse{
			Code:    http.StatusConflict,
			Message: "secret already exists in the cluster",
		})

		return
	} else if err != nil {
		errorHandler.Handle(emperror.With(
			emperror.Wrap(err, "failed to install secret into cluster"),
			"clusterId", commonCluster.GetID(),
			"organizationId", commonCluster.GetOrganizationId(),
			"secret", secretName,
			"sourceSecret", secretRequest.SourceSecretName,
		))

		c.AbortWithStatusJSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error installing secret into cluster",
			Error:   err.Error(),
		})

		return
	}

	response := InstallSecretResponse{
		Name: secretName,
	}

	c.JSON(http.StatusOK, response)
}

// MergeSecretInCluster installs a particular secret to a cluster's namespace.
func MergeSecretInCluster(c *gin.Context) {
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
		SourceSecretName: request.SourceSecretName,
		Namespace:        request.Namespace,
		Spec:             map[string]cluster.InstallSecretRequestSpecItem{},
	}

	for key, spec := range request.Spec {
		secretRequest.Spec[key] = cluster.InstallSecretRequestSpecItem{
			Source:    spec.Source,
			SourceMap: spec.SourceMap,
			Value:     spec.Value,
		}
	}

	secretName := c.Param("secretName")

	// Either spec is not defined (empty) or at least one spec is not a value or empty
	needsSecret := len(secretRequest.Spec) == 0
	for _, spec := range secretRequest.Spec {
		if spec.Source != "" || len(spec.SourceMap) != 0 || spec.Value == "" {
			needsSecret = true
			break
		}
	}

	// If there is no separate pipeline secret name use the same as the cluster request name
	if needsSecret && secretRequest.SourceSecretName == "" {
		secretRequest.SourceSecretName = secretName
	}

	err := cluster.MergeSecret(commonCluster, secretName, secretRequest)

	if err == cluster.ErrSecretNotFound {
		ginutils.ReplyWithErrorResponse(c, &pkgCommon.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "secret not found",
		})

		return
	} else if err == cluster.ErrKubernetesSecretNotFound {
		ginutils.ReplyWithErrorResponse(c, &pkgCommon.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "kubernetes secret not found",
		})

		return
	} else if err != nil {
		errorHandler.Handle(emperror.With(
			emperror.Wrap(err, "failed to install secret into cluster"),
			"clusterId", commonCluster.GetID(),
			"organizationId", commonCluster.GetOrganizationId(),
			"secret", secretName,
			"sourceSecret", secretRequest.SourceSecretName,
		))

		c.AbortWithStatusJSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error installing secret into cluster",
			Error:   err.Error(),
		})

		return
	}

	response := InstallSecretResponse{
		Name: secretName,
	}

	c.JSON(http.StatusOK, response)
}
