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

package ginutils

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

type contextKey int

const (
	paramsKey contextKey = iota
)

// HTTPHandlerToGinHandlerFunc wraps a http.Handler so that it works as a gin.HandlerFunc
func HTTPHandlerToGinHandlerFunc(handler http.Handler) gin.HandlerFunc {
	return HTTPHandlerFuncToGinHandlerFunc(handler.ServeHTTP)
}

// HTTPHandlerFuncToGinHandlerFunc wraps a http.HandlerFunc so that it works as a gin.HandlerFunc
func HTTPHandlerFuncToGinHandlerFunc(handlerFunc http.HandlerFunc) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := ctx.Request
		handlerFunc(ctx.Writer, req.WithContext(context.WithValue(req.Context(), paramsKey, ctx.Params)))
	}
}

// GetParams return the route params from the given context
func GetParams(ctx context.Context) gin.Params {
	if ginctx, ok := ctx.(*gin.Context); ok {
		return ginctx.Params
	}

	if params, ok := ctx.Value(paramsKey).(gin.Params); ok {
		return params
	}

	return nil
}
