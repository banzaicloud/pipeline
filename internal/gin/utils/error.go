package ginutils

import (
	"github.com/banzaicloud/pipeline/pkg/common"
	"github.com/gin-gonic/gin"
)

// ReplyWithErrorResponse replies with an error response.
func ReplyWithErrorResponse(ctx *gin.Context, errorResponse *common.ErrorResponse) {
	ctx.JSON(errorResponse.Code, errorResponse)
}
