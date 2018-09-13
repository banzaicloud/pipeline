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
