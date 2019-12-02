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
	"context"
	"net/http"

	"emperror.dev/emperror"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/global"
	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	"github.com/banzaicloud/pipeline/pkg/brn"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/pkg/kubernetes/custom/npls"
	"github.com/banzaicloud/pipeline/src/api/common"
	"github.com/banzaicloud/pipeline/src/cluster"
)

// NodePoolManagerAPI implements the Node pool Label Management API actions.
type NodepoolManagerAPI struct {
	clusterGetter common.ClusterGetter
	clientFactory DynamicClientFactory

	logger       logrus.FieldLogger
	errorHandler emperror.Handler
}

// NewNodepoolManagerAPI returns a new NodepoolManagerAPI instance.
func NewNodepoolManagerAPI(
	clusterGetter common.ClusterGetter,
	clientFactory DynamicClientFactory,
	logger logrus.FieldLogger,
	errorHandler emperror.Handler,
) *NodepoolManagerAPI {
	return &NodepoolManagerAPI{
		clusterGetter: clusterGetter,
		clientFactory: clientFactory,
		logger:        logger,
		errorHandler:  errorHandler,
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

	manager := npls.NewManager(client, global.Config.Cluster.Namespace)

	sets, err := manager.GetAll()
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
				Reserved: cluster.IsReservedDomainKey(labelKey),
			})
		}
		response[npName] = labels
	}

	c.JSON(http.StatusOK, response)
}

func (n *NodepoolManagerAPI) SetNodepoolLabelSets(c *gin.Context) {
	ctx := ginutils.Context(context.Background(), c)

	var nodepoolLabelSets map[string]map[string]string
	if err := c.BindJSON(&nodepoolLabelSets); err != nil {
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}

	commonCluster, ok := n.clusterGetter.GetClusterFromRequest(c)
	if !ok {
		return
	}

	ready, err := commonCluster.IsReady()
	if err != nil {
		err = errors.WithMessage(err, "failed to check cluster readiness")
		errorHandler.Handle(err)

		ginutils.ReplyWithErrorResponse(c, ErrorResponseFrom(err))
	} else if !ready {
		err := errors.New("cluster is not ready")
		errorHandler.Handle(err)

		ginutils.ReplyWithErrorResponse(c, &pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "unable to set node pool labels",
			Error:   err.Error(),
		})
		return
	}

	updatedNodePools, err := getNodePoolsWithUpdatedLabels(commonCluster, nodepoolLabelSets)
	if err != nil {
		errorHandler.Handle(err)
		ginutils.ReplyWithErrorResponse(c, ErrorResponseFrom(err))
		return
	}

	labelsMap, err := cluster.GetDesiredLabelsForCluster(ctx, commonCluster, updatedNodePools, false)
	if err != nil {
		errorHandler.Handle(err)
		ginutils.ReplyWithErrorResponse(c, ErrorResponseFrom(err))
		return
	}

	secretID := brn.New(commonCluster.GetOrganizationId(), brn.SecretResourceType, commonCluster.GetConfigSecretId()).String()
	client, err := n.clientFactory.FromSecret(c.Request.Context(), secretID)
	if err != nil {
		errorHandler.Handle(err)

		ginutils.ReplyWithErrorResponse(c, ErrorResponseFrom(err))
		return
	}

	manager := npls.NewManager(client, global.Config.Cluster.Namespace)

	err = manager.Sync(labelsMap)
	if err != nil {
		type errorCollection interface {
			Errors() []error
		}
		if _, ok := err.(errorCollection); ok {
			err = pkgErrors.NewMultiErrorWithFormatter(err)
		}
		errorHandler.Handle(err)
		ginutils.ReplyWithErrorResponse(c, ErrorResponseFrom(err))
		return
	}

	c.JSON(http.StatusOK, "")
}

// getNodePoolsWithUpdatedLabels returns NodePoolStatus map with updated user labels from NodePoolLabelSets
func getNodePoolsWithUpdatedLabels(commonCluster cluster.CommonCluster, nodepoolLabelSets map[string]map[string]string) (map[string]*pkgCluster.NodePoolStatus, error) {

	nodePools := make(map[string]*pkgCluster.NodePoolStatus)
	clusterStatus, err := commonCluster.GetStatus()
	if err != nil {
		return nodePools, err
	}
	for nodePoolName, np := range clusterStatus.NodePools {
		if labelSet, ok := nodepoolLabelSets[nodePoolName]; ok {
			err := pkgCommon.ValidateNodePoolLabels(labelSet)
			if err != nil {
				return nil, err
			}
			np.Labels = labelSet
			nodePools[nodePoolName] = np
		}
	}

	return nodePools, nil
}
