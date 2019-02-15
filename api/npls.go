package api

import (
	"net/http"

	"github.com/banzaicloud/nodepool-labels-operator/pkg/npls"
	"github.com/banzaicloud/pipeline/cluster"
	pipConfig "github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// returns NodePoolStatus map with updated user labels from NodePoolLabelSets
func getNodePoolsWithUpdatedLabels(commonCluster cluster.CommonCluster, nodepoolLabelSets npls.NodepoolLabelSets) (map[string]*pkgCluster.NodePoolStatus, error) {

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

func SetNodepoolLabelSets(c *gin.Context) {
	var nodepoolLabelSets npls.NodepoolLabelSets
	if err := c.BindJSON(&nodepoolLabelSets); err != nil {
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})

		return
	}

	commonCluster, ok := getClusterFromRequest(c)
	if !ok {
		return
	}

	type errorCollection interface {
		Errors() []error
	}

	updatedNodePools, err := getNodePoolsWithUpdatedLabels(commonCluster, nodepoolLabelSets)
	if err != nil {
		errorHandler.Handle(err)
		ginutils.ReplyWithErrorResponse(c, errorResponseFrom(err))
		return
	}

	err = cluster.DeployNodePoolLabelsSet(commonCluster, updatedNodePools, false)
	if err != nil {
		if errs, ok := err.(errorCollection); ok {
			for _, e := range errs.Errors() {
				errorHandler.Handle(e)
			}
		}
		errorHandler.Handle(err)
		ginutils.ReplyWithErrorResponse(c, errorResponseFrom(err))
		return
	}

	c.JSON(http.StatusOK, "")
}

func GetNodepoolLabelSets(c *gin.Context) {
	commonCluster, ok := getClusterFromRequest(c)
	if !ok {
		return
	}

	config, err := commonCluster.GetK8sConfig()
	if err != nil {
		ginutils.ReplyWithErrorResponse(c, errorResponseFrom(err))
		return
	}

	k8sconfig, err := k8sclient.NewClientConfig(config)
	if err != nil {
		ginutils.ReplyWithErrorResponse(c, errorResponseFrom(err))
		return
	}
	pipelineSystemNamespace := viper.GetString(pipConfig.PipelineSystemNamespace)
	m, err := npls.NewNPLSManager(k8sconfig, pipelineSystemNamespace)
	if err != nil {
		ginutils.ReplyWithErrorResponse(c, errorResponseFrom(err))
		return
	}

	sets, err := m.GetAll()
	if err != nil {
		ginutils.ReplyWithErrorResponse(c, errorResponseFrom(err))
		return
	}

	c.JSON(http.StatusOK, sets)
}
