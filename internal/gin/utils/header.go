package ginutils

import (
	"fmt"
	"net/http"

	"github.com/banzaicloud/pipeline/pkg/common"
	"github.com/gin-gonic/gin"
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
