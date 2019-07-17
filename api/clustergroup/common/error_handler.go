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

package common

import (
	"net/http"

	"emperror.dev/emperror"
	"github.com/gin-gonic/gin"

	"github.com/banzaicloud/pipeline/api"
	cgroup "github.com/banzaicloud/pipeline/internal/clustergroup"
	"github.com/banzaicloud/pipeline/internal/clustergroup/deployment"
	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
)

type ErrorHandler struct {
	Handler emperror.Handler
}

func (e ErrorHandler) Handle(c *gin.Context, err error) {
	ginutils.ReplyWithErrorResponse(c, e.errorResponseFrom(err))

	e.Handler.Handle(err)
}

// errorResponseFrom translates the given error into a components.ErrorResponse
func (e ErrorHandler) errorResponseFrom(err error) *pkgCommon.ErrorResponse {
	if e, ok := err.(*gin.Error); ok {
		if e.IsType(gin.ErrorTypeBind) {
			return &pkgCommon.ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: "Error parsing request",
				Error:   err.Error(),
			}
		}
	}

	var code int
	if cgroup.IsClusterGroupNotFoundError(err) || deployment.IsDeploymentNotFoundError(err) || cgroup.IsFeatureRecordNotFoundError(err) {
		code = http.StatusNotFound
	} else if cgroup.IsClusterGroupAlreadyExistsError(err) || cgroup.IsUnableToJoinMemberClusterError(err) || cgroup.IsInvalidClusterGroupCreateRequestError(err) || cgroup.IsClusterGroupUpdateRejectedError(err) {
		code = http.StatusBadRequest
	}

	if code > 0 {
		return &pkgCommon.ErrorResponse{
			Code:    code,
			Error:   err.Error(),
			Message: err.Error(),
		}
	}

	if err, ok := cgroup.IsMemberClusterNotFoundError(err); ok {
		return &pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Error:   err.Error(),
			Message: err.Message(),
		}
	}

	if err, ok := cgroup.IsMemberClusterPartOfAClusterGroupError(err); ok {
		return &pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Error:   err.Error(),
			Message: err.Message(),
		}
	}

	return api.ErrorResponseFrom(err)
}
