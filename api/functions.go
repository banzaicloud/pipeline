package api

import (
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/gin-gonic/gin"
	"net/http"
)

// Apply the given functions to the cluster
func ApplyFunctions(c *gin.Context) {

}

// List available functions to apply on clusters
func ListFunctions(c *gin.Context) {
	var functionList []string
	for k := range cluster.HookMap {
		functionList = append(functionList, k)
	}
	c.JSON(http.StatusOK, functionList)
	return
}
