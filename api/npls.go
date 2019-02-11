package api

import (
	"net/http"

	"github.com/banzaicloud/nodepool-labels-operator/pkg/npls"
	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/gin-gonic/gin"
)

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

	m, err := npls.NewNPLSManager(k8sconfig, "pipeline-infra")
	if err != nil {
		ginutils.ReplyWithErrorResponse(c, errorResponseFrom(err))
		return
	}

	type errorCollection interface {
		Errors() []error
	}

	err = m.Sync(nodepoolLabelSets)
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

	c.JSON(http.StatusOK, nil)
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

	m, err := npls.NewNPLSManager(k8sconfig, "pipeline-infra")
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
