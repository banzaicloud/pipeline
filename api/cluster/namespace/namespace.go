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

package namespace

import (
	"emperror.dev/emperror"
	"github.com/gin-gonic/gin"

	"github.com/banzaicloud/pipeline/api/common"
)

type API struct {
	clusterGetter common.ClusterGetter
	errorHandler  emperror.Handler
}

func NewAPI(clusterGetter common.ClusterGetter, errorHandler emperror.Handler) *API {
	return &API{
		clusterGetter: clusterGetter,
		errorHandler:  errorHandler,
	}
}

func (a *API) RegisterRoutes(r gin.IRouter) {
	r.DELETE("", a.Delete)
	r.GET("", a.List)
}
