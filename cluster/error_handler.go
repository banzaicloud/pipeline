package cluster

import (
	"github.com/banzaicloud/pipeline/config"
	"github.com/goph/emperror"
)

var errorHandler emperror.Handler

func init() {
	errorHandler = config.ErrorHandler()
}
