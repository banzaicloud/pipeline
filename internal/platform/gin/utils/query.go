package ginutils

import (
	"fmt"
	"net/http"

	"github.com/banzaicloud/pipeline/pkg/common"
	"github.com/gin-gonic/gin"
)

// RequiredQuery returns a query value or responds with an error.
func RequiredQuery(ctx *gin.Context, queryName string) (string, bool) {
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
