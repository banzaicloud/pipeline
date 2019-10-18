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
		fnCtx := map[string]interface{}{"orgiD": orgIDStr, "clusterID": clusterIDStr}

		ap.logger.Debug("proxying to anchore", fnCtx)

		clusterID, err := strconv.ParseUint(clusterIDStr, 0, 64)
		if err != nil {
			ap.logger.Warn("failed to parse the cluster id", fnCtx)

			c.JSON(http.StatusInternalServerError, c.AbortWithError(http.StatusInternalServerError, err))
			return
		}

		config, err := ap.configService.GetConfiguration(c.Request.Context(), uint(clusterID))
		if err != nil {
			ap.logger.Warn("failed to retrieve anchore configuraiton", fnCtx)

			c.JSON(http.StatusInternalServerError, c.AbortWithError(http.StatusInternalServerError, err))
			return
		}

		backendURL, err := url.Parse(config.Endpoint)
		if err != nil {
			ap.logger.Warn("failed to parse the backend URL", fnCtx)

			c.JSON(http.StatusInternalServerError, c.AbortWithError(http.StatusInternalServerError, err))
			return
		}

		username, password, err := ap.processCredentials(c.Request.Context(), config, uint(clusterID))
		if err != nil {
			ap.logger.Warn("failed to process anchore credentials", fnCtx)

			c.JSON(http.StatusInternalServerError, c.AbortWithError(http.StatusInternalServerError, err))
			return
		}

		director := func(r *http.Request) {

			r.Host = backendURL.Host // this is a must!
			r.Header.Set("User-Agent", "Pipeline/go")
			r.SetBasicAuth(username, password)

			r.URL.Scheme = backendURL.Scheme
			r.URL.Host = backendURL.Host
			r.URL.Path = strings.Join([]string{backendURL.Path, ap.getProxyPath(r.URL.Path, orgIDStr, clusterIDStr)}, "/")

			if backendURL.RawQuery == "" || r.URL.RawQuery == "" {
				r.URL.RawQuery = backendURL.RawQuery + r.URL.RawQuery
			} else {
				r.URL.RawQuery = backendURL.RawQuery + "&" + r.URL.RawQuery
			}
		}

		modifyResponse := func(*http.Response) error {
			// todo implement me
			return nil
		}

		proxy := &httputil.ReverseProxy{
			Director:       director,
			ModifyResponse: modifyResponse,
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// getProxyPath processes the path the request is proxied to
func (ap AnchoreProxy) getProxyPath(sourcePath string, orgID string, clusterID string) string {
	prefixToTrim := fmt.Sprintf("%s/api/v1/orgs/%s/clusters/%s/", ap.basePath, orgID, clusterID)

	return ap.adaptImageScanResourcePath(strings.TrimPrefix(sourcePath, prefixToTrim))
}

// adaptImageScanResourcePath adapts the pipeline resources to the anchore ones
func (ap AnchoreProxy) adaptImageScanResourcePath(proxyPath string) string {

	// pipeline resource -> anchore resource
	pathaDaptors := map[string]string{
		"imagescan": "images",
	}

	adaptedPath := proxyPath
	for pipelineResource, anchorResource := range pathaDaptors {
		if strings.Contains(proxyPath, pipelineResource) {
			adaptedPath = strings.Replace(proxyPath, pipelineResource, anchorResource, 1)
		}
	}

	return adaptedPath
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
