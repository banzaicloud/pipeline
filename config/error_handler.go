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

package config

import (
	"fmt"
	"sync"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	logrushandler "emperror.dev/handler/logrus"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/banzaicloud/pipeline/internal/platform/log"
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

	loggerHandler := logrushandler.New(logger)

	return emperror.HandlerFunc(func(err error) {
		var stackTracer interface{ StackTrace() errors.StackTrace }
		if errors.As(err, &stackTracer) {
			stackTrace := stackTracer.StackTrace()

			if len(stackTrace) > 0 {
				frame := stackTrace[0]

				err = emperror.With(
					err,
					"func", fmt.Sprintf("%n", frame), // nolint: govet
					"file", fmt.Sprintf("%v", frame), // nolint: govet
				)
			}
		}

		loggerHandler.Handle(err)
	})
}
