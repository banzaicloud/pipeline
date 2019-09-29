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

// RequiredQueryOrAbort returns a query value or responds with an error.
func RequiredQueryOrAbort(ctx *gin.Context, queryName string) (string, bool) {
	value := ctx.Query(queryName)
	if len(value) == 0 {
		ReplyWithErrorResponse(ctx, RequiredQueryMissingErrorResponse(queryName))

		return "", false
	}

	return value, true
}

// RequiredQueryMissingErrorResponse creates a common.ErrorResponse denoting missing required header.
func RequiredQueryMissingErrorResponse(queryName string) *common.ErrorResponse {
	return &common.ErrorResponse{
		Code:    http.StatusBadRequest,
		Error:   "Query parameter required",
		Message: fmt.Sprintf("Required query parameter '%s' is missing", queryName),
	}
}
