package config

import (
	"fmt"
	"sync"

	"github.com/banzaicloud/pipeline/internal/platform/log"
	"github.com/goph/emperror"
	"github.com/goph/emperror/errorlogrus"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var errorHandler emperror.Handler
var errorHandlerOnce sync.Once

// ErrorHandler returns an error handler.
func ErrorHandler() emperror.Handler {
	errorHandlerOnce.Do(func() {
		errorHandler = newErrorHandler()
	})

	return errorHandler
}

func newErrorHandler() emperror.Handler {
	logger := log.NewLogger(log.Config{
		Level:  logrus.ErrorLevel.String(),
		Format: viper.GetString("logging.logformat"),
	})

	loggerHandler := errorlogrus.NewHandler(logger)

	return emperror.HandlerFunc(func(err error) {
		if stackTrace, ok := emperror.StackTrace(err); ok && len(stackTrace) > 0 {
			frame := stackTrace[0]

			err = emperror.With(
				err,
				"func", fmt.Sprintf("%n", frame), // nolint: govet
				"file", fmt.Sprintf("%s", frame), // nolint: govet
				"line", fmt.Sprintf("%d", frame),
			)
		}

		loggerHandler.Handle(err)
	})
}
