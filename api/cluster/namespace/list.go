// Copyright © 2019 Banzai Cloud
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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

func (a *API) List(c *gin.Context) {
	cluster, ok := a.clusterGetter.GetClusterFromRequest(c)
	if !ok {
		return
	}

	// TODO: factor out to a common method
	kubeConfig, err := cluster.GetK8sConfig()
	if err != nil {
		a.errorHandler.Handle(errors.Wrap(err, "failed to get kube config"))

		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting kubeconfig",
			Error:   err.Error(),
		})
		return
	}

	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		a.errorHandler.Handle(errors.Wrap(err, "failed to get kube client"))

		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting kube client",
			Error:   err.Error(),
		})
		return
	}

	nsList, err := client.CoreV1().Namespaces().List(v1.ListOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		a.errorHandler.Handle(errors.Wrap(err, "failed to list namespaces"))

		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "error listing namespaces",
			Error:   err.Error(),
		})
		return
	}

	type nsItem struct {
		Name string
	}

	var namespaces = make([]nsItem, 0, len(nsList.Items))
	for _, ns := range nsList.Items {
		namespaces = append(namespaces, nsItem{Name: ns.Name})
	}

	c.JSON(http.StatusOK, namespaces)
}
