package ginutils

import (
	"net/http"
	"strconv"

	"github.com/banzaicloud/pipeline/pkg/common"
	"github.com/gin-gonic/gin"
)

// UintParam returns a parameter parsed as uint or responds with an error.
func UintParam(ctx *gin.Context, paramName string) (uint, bool) {
	value := ctx.Param(paramName)
	uintValue, err := strconv.ParseUint(value, 10, 0)
	if err != nil {
		ReplyWithErrorResponse(ctx, &common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Error:   "Invalid parameter",
			Message: "Parameter 'id' must be a positive, numeric value",
		})

		return 0, false
	}

	return uint(uintValue), true
}
