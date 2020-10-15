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

package api

import (
	"net/http"

	"emperror.dev/emperror"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/global"
	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/kubernetes/custom/npls"
	"github.com/banzaicloud/pipeline/pkg/sdk/brn"
	"github.com/banzaicloud/pipeline/src/api/common"
)

// NodePoolManagerAPI implements the Node pool Label Management API actions.
type NodepoolManagerAPI struct {
	clusterGetter  common.ClusterGetter
	clientFactory  common.DynamicClientFactory
	labelValidator LabelValidator

	logger       logrus.FieldLogger
	errorHandler emperror.Handler
}

// LabelValidator validates Kubernetes object labels.
type LabelValidator interface {
	// ValidateKey validates a label key.
	ValidateKey(key string) error

	// ValidateValue validates a label value.
	ValidateValue(value string) error
}

// NewNodepoolManagerAPI returns a new NodepoolManagerAPI instance.
func NewNodepoolManagerAPI(
	clusterGetter common.ClusterGetter,
	clientFactory common.DynamicClientFactory,
	labelValidator LabelValidator,
	logger logrus.FieldLogger,
	errorHandler emperror.Handler,
) *NodepoolManagerAPI {
	return &NodepoolManagerAPI{
		clusterGetter:  clusterGetter,
		clientFactory:  clientFactory,
		labelValidator: labelValidator,
		logger:         logger,
		errorHandler:   errorHandler,
	}
}

func (n *NodepoolManagerAPI) GetNodepoolLabelSets(c *gin.Context) {
	response := make(map[string][]pkgCluster.NodePoolLabel)

	commonCluster, ok := n.clusterGetter.GetClusterFromRequest(c)
	if !ok {
		return
	}

	errorHandler = emperror.HandlerWith(
		errorHandler,
		"clusterId", commonCluster.GetID(),
		"cluster", commonCluster.GetName(),
	)

	ready, err := commonCluster.IsReady()
	if err != nil {
		err = errors.WithMessage(err, "failed to check if the cluster is ready")
		errorHandler.Handle(err)
	}
	if err != nil || !ready { // Cluster is not ready yet or we can't check if it's ready
		c.JSON(http.StatusPartialContent, response)
		return
	}

	secretID := brn.New(commonCluster.GetOrganizationId(), brn.SecretResourceType, commonCluster.GetConfigSecretId()).String()
	client, err := n.clientFactory.FromSecret(c.Request.Context(), secretID)
	if err != nil {
		errorHandler.Handle(err)

		ginutils.ReplyWithErrorResponse(c, ErrorResponseFrom(err))
		return
	}

	manager := npls.NewManager(client, global.Config.Cluster.Labels.Namespace)

	sets, err := manager.GetAll(c.Request.Context())
	if err != nil {
		errorHandler.Handle(err)

		ginutils.ReplyWithErrorResponse(c, ErrorResponseFrom(err))
		return
	}

	for npName, labelMap := range sets {
		labels := make([]pkgCluster.NodePoolLabel, 0, len(labelMap))
		for labelKey, labelValue := range labelMap {
			labels = append(labels, pkgCluster.NodePoolLabel{
				Name:     labelKey,
				Value:    labelValue,
				Reserved: n.labelValidator.ValidateKey(labelKey) != nil, // TODO: extract reserved logic
			})
		}
		response[npName] = labels
	}

	c.JSON(http.StatusOK, response)
}
