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
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/banzaicloud/pipeline/internal/common"
	anchore "github.com/banzaicloud/pipeline/internal/security"
)

type AnchoreProxy struct {
	basePath      string
	configService anchore.ConfigurationService
	secretStore   common.SecretStore
	logger        common.Logger
}

func NewAnchoreProxy(basePath string, configurationService anchore.ConfigurationService, secretStore common.SecretStore, logger common.Logger) AnchoreProxy {

	return AnchoreProxy{
		basePath:      basePath,
		configService: configurationService,
		secretStore:   secretStore,
		logger:        logger,
	}
}

func (ap AnchoreProxy) Proxy() gin.HandlerFunc {

	return func(c *gin.Context) {

		orgIDStr := c.Param("orgid")
		clusterIDStr := c.Param("id")

		clusterID, err := strconv.ParseUint(clusterIDStr, 0, 64)
		if err != nil {
			c.JSON(http.StatusInternalServerError, c.AbortWithError(http.StatusInternalServerError, err))
			return
		}

		config, err := ap.configService.GetConfiguration(c.Request.Context(), uint(clusterID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, c.AbortWithError(http.StatusInternalServerError, err))
			return
		}

		backendURL, err := url.Parse(config.Endpoint)
		if err != nil {
			c.JSON(http.StatusInternalServerError, c.AbortWithError(http.StatusInternalServerError, err))
			return
		}

		username, password, err := ap.processCredentials(c.Request.Context(), config, uint(clusterID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, c.AbortWithError(http.StatusInternalServerError, err))
			return
		}

		director := func(r *http.Request) {
			targetQuery := backendURL.RawQuery
			r.SetBasicAuth(username, password)
			r.URL.Scheme = backendURL.Scheme
			r.URL.Host = backendURL.Host
			r.URL.Path = strings.Join([]string{backendURL.Path, ap.getProxyPath(r.URL.Path, orgIDStr, clusterIDStr)}, "/")

			r.Host = backendURL.Host // this is a must!

			if targetQuery == "" || r.URL.RawQuery == "" {
				r.URL.RawQuery = targetQuery + r.URL.RawQuery
			} else {
				r.URL.RawQuery = targetQuery + "&" + r.URL.RawQuery
			}

			r.Header.Set("User-Agent", "Pipeline/go")
		}

		proxy := &httputil.ReverseProxy{Director: director}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func (ap AnchoreProxy) getProxyPath(sourcePath string, orgID string, clusterID string) string {
	prefixToTrim := fmt.Sprintf("%s/api/v1/orgs/%s/clusters/%s/", ap.basePath, orgID, clusterID)

	return strings.TrimPrefix(sourcePath, prefixToTrim)
}

// processCredentials depending on the configuration get the appropriate credentials for accessing anchore
func (ap AnchoreProxy) processCredentials(ctx context.Context, config anchore.Config, clusterID uint) (string, string, error) {

	if config.UserSecret != "" { // custom anchore
		return anchore.GetCustomAnchoreCredentials(ctx, ap.secretStore, config.UserSecret, ap.logger)
	}

	// managed anchore, generated user
	username := anchore.GetUserName(clusterID)
	password, err := anchore.GetUserSecret(ctx, ap.secretStore, username, ap.logger)

	return username, password, err
}
