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

package ginutils

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/banzaicloud/pipeline/pkg/common"
)

// GetRequiredHeader returns a header value or responds with an error.
func GetRequiredHeader(ctx *gin.Context, headerName string) (string, bool) {
	value := ctx.GetHeader(headerName)
	if len(value) == 0 {
		ReplyWithErrorResponse(ctx, RequiredHeaderMissingErrorResponse(headerName))

		return "", false
	}

	return value, true
}

// RequiredHeaderMissingErrorResponse creates a common.ErrorResponse denoting missing required header.
func RequiredHeaderMissingErrorResponse(headerName string) *common.ErrorResponse {
	return &common.ErrorResponse{
		Code:    http.StatusBadRequest,
		Error:   "Header parameter required",
		Message: fmt.Sprintf("Required header parameter '%s' is missing", headerName),
	}
}
