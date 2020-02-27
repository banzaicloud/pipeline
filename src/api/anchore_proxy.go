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
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"emperror.dev/errors"
	"github.com/gin-gonic/gin"

	"github.com/banzaicloud/pipeline/internal/anchore"
	"github.com/banzaicloud/pipeline/internal/common"
)

const pipelineUserAgent = "Pipeline/go"

type AnchoreProxy struct {
	basePath       string
	configProvider anchore.ConfigProvider

	errorHandler common.ErrorHandler
	logger       common.Logger
}

func NewAnchoreProxy(
	basePath string,
	configProvider anchore.ConfigProvider,

	errorHandler common.ErrorHandler,
	logger common.Logger,
) AnchoreProxy {
	return AnchoreProxy{
		basePath:       basePath,
		configProvider: configProvider,

		errorHandler: errorHandler,
		logger:       logger,
	}
}

func (ap AnchoreProxy) Proxy() gin.HandlerFunc {
	return func(c *gin.Context) {
		ap.logger.Info("proxying to anchore")

		orgID, err := ap.idFromPath(c, "orgid")
		if err != nil {
			ap.errorHandler.HandleContext(c.Request.Context(), err)

			c.JSON(http.StatusInternalServerError, c.AbortWithError(http.StatusInternalServerError, err))
			return
		}

		clusterID, err := ap.idFromPath(c, "id")
		if err != nil {
			ap.errorHandler.HandleContext(c.Request.Context(), err)

			c.JSON(http.StatusInternalServerError, c.AbortWithError(http.StatusInternalServerError, err))
			return
		}

		proxyPath := c.Param("proxyPath")

		proxy, err := ap.buildReverseProxy(c.Request.Context(), proxyPath, orgID, clusterID)
		if err != nil {
			ap.errorHandler.HandleContext(c.Request.Context(), err)

			c.JSON(http.StatusInternalServerError, c.AbortWithError(http.StatusInternalServerError, err))
			return
		}

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func (ap AnchoreProxy) buildProxyDirector(ctx context.Context, proxyPath string, orgID uint, clusterID uint) (func(req *http.Request), error) {
	fnCtx := map[string]interface{}{"orgiD": orgID, "clusterID": clusterID}
	config, err := ap.configProvider.GetConfiguration(ctx, clusterID)
	if err != nil {
		ap.logger.Warn("failed to retrieve anchore configuration", fnCtx)

		return nil, errors.WrapIf(err, "failed to retrieve anchore configuration")
	}

	backendURL, err := url.Parse(config.Endpoint)
	if err != nil {
		ap.logger.Warn("failed to parse the backend URL", fnCtx)

		return nil, errors.WrapIf(err, "failed to parse the backend URL")
	}

	return func(r *http.Request) {
		r.Host = backendURL.Host // this is a must!
		r.Header.Set("User-Agent", pipelineUserAgent)
		r.SetBasicAuth(config.User, config.Password)

		r.URL.Scheme = backendURL.Scheme
		r.URL.Host = backendURL.Host
		r.URL.Path = strings.Join([]string{backendURL.Path, proxyPath}, "")

		if backendURL.RawQuery == "" || r.URL.RawQuery == "" {
			r.URL.RawQuery = backendURL.RawQuery + r.URL.RawQuery
		} else {
			r.URL.RawQuery = backendURL.RawQuery + "&" + r.URL.RawQuery
		}
	}, nil
}

func (ap AnchoreProxy) buildProxyModifyResponseFunc(ctx context.Context) (func(*http.Response) error, error) {
	return func(resp *http.Response) error {
		// handle individual error codes here if required
		if resp.StatusCode != http.StatusOK {
			return errors.Errorf("error received from Anchore ( StatusCode: %d, Status: %s )", resp.StatusCode, resp.Status)
		}
		return nil
	}, nil
}

func (ap AnchoreProxy) buildReverseProxy(ctx context.Context, proxyPath string, orgID uint, clusterID uint) (*httputil.ReverseProxy, error) {
	director, err := ap.buildProxyDirector(ctx, proxyPath, orgID, clusterID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to build reverse proxy")
	}

	modifyResponse, err := ap.buildProxyModifyResponseFunc(ctx)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to build reverse proxy")
	}

	errorHandler := func(rw http.ResponseWriter, req *http.Request, err error) {
		rw.WriteHeader(http.StatusInternalServerError)
		if _, err := rw.Write([]byte(err.Error())); err != nil {
			ap.logger.Error("failed to write error response body")
		}
	}

	proxy := &httputil.ReverseProxy{
		Director:       director,
		ModifyResponse: modifyResponse,
		ErrorHandler:   errorHandler,
	}

	return proxy, nil
}

func (ap AnchoreProxy) idFromPath(c *gin.Context, paramKey string) (uint, error) {
	id, err := strconv.ParseUint(c.Param(paramKey), 0, 64)
	if err != nil {
		return 0, errors.WrapIf(err, "failed to get id from request path")
	}

	return uint(id), nil
}
