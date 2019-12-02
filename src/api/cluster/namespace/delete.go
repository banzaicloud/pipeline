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

package namespace

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/pkg/brn"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
)

// Delete deletes a kuberenetes namespace.
func (a *API) Delete(c *gin.Context) {
	cluster, ok := a.clusterGetter.GetClusterFromRequest(c)
	if !ok {
		return
	}

	secretID := brn.New(cluster.GetOrganizationId(), brn.SecretResourceType, cluster.GetConfigSecretId()).String()
	client, err := a.clientFactory.FromSecret(c.Request.Context(), secretID)
	if err != nil {
		a.errorHandler.Handle(err)

		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting kube client",
			Error:   err.Error(),
		})
		return
	}

	err = client.CoreV1().Namespaces().Delete(c.Param("namespace"), &meta_v1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		a.errorHandler.Handle(errors.Wrap(err, "failed to delete namespace"))

		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error deleting namespace",
			Error:   err.Error(),
		})
		return
	}

	c.Status(http.StatusAccepted)
}
