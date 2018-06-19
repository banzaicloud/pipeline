package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// MetaHandler lists routes with their available methods
func MetaHandler(router *gin.Engine, subpath string) gin.HandlerFunc {
	routes := map[string][]string{}
	for _, route := range router.Routes() {
		if strings.HasPrefix(route.Path, subpath) {
			routes[route.Path] = append(routes[route.Path], route.Method)
		}
	}
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, routes)
	}
}
