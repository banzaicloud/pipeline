package api

import (
	"net/http"
	"strconv"

	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/config"
	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/gin-gonic/gin"
)

// DeleteClusterResponse describes Pipeline's DeleteCluster API response
type DeleteClusterResponse struct {
	Status     int    `json:"status"`
	Name       string `json:"name"`
	Message    string `json:"message"`
	ResourceID uint   `json:"id"`
}

// DeleteCluster deletes a K8S cluster from the cloud
func DeleteCluster(c *gin.Context) {
	commonCluster, ok := getClusterFromRequest(c)
	if ok != true {
		return
	}

	force, _ := strconv.ParseBool(c.DefaultQuery("force", "false"))

	// DeleteCluster deletes the underlying model, so we get this data here
	clusterID, clusterName := commonCluster.GetID(), commonCluster.GetName()

	// TODO: move these to a struct and create them only once upon application init
	clusters := intCluster.NewClusters(config.DB())
	secretValidator := providers.NewSecretValidator(secret.Store)
	clusterManager := cluster.NewManager(clusters, secretValidator, log, errorHandler)

	ctx := ginutils.Context(c.Request.Context(), c)

	clusterManager.DeleteCluster(ctx, commonCluster, force, &kubeProxyCache)

	c.JSON(http.StatusAccepted, DeleteClusterResponse{
		Status:     http.StatusAccepted,
		Name:       clusterName,
		ResourceID: clusterID,
	})
}
