package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func MetaHandler(router *gin.Engine, subpath string) gin.HandlerFunc {
	routes := []string{}
	for _, route := range router.Routes() {
		if strings.HasPrefix(route.Path, subpath) {
			routes = append(routes, route.Path)
		}
	}
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, routes)
	}
}
