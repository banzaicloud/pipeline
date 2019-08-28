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

package global

import (
	"sync"

	"emperror.dev/emperror"
	logurhandler "emperror.dev/handler/logur"
	"github.com/sirupsen/logrus"
	logrusadapter "logur.dev/adapter/logrus"
)

var errorHandler emperror.Handler
var errorHandlerOnce sync.Once
var errorHandlerMu sync.RWMutex
var errorHandlerSubscribers []func(h emperror.Handler)

// ErrorHandler returns an error handler.
func ErrorHandler() emperror.Handler {
	errorHandlerMu.RLock()
	defer errorHandlerMu.RUnlock()

	errorHandlerOnce.Do(func() {
		errorHandler = newErrorHandler()
	})

	return errorHandler
}

func newErrorHandler() emperror.Handler {
	return logurhandler.WithStackInfo(logurhandler.New(logrusadapter.New(logrus.New())))
}

// SubscribeErrorHandler subscribes a handler for global error handler changes.
func SubscribeErrorHandler(s func(h emperror.Handler)) {
	errorHandlerMu.Lock()
	defer errorHandlerMu.Unlock()

	errorHandlerSubscribers = append(errorHandlerSubscribers, s)
}

// SetErrorHandler sets a global error handler.
//
// Note: setting an error handler after the application bootstrap is not safe.
func SetErrorHandler(h emperror.Handler) {
	errorHandlerMu.Lock()
	defer errorHandlerMu.Unlock()

	errorHandler = h

	for _, s := range errorHandlerSubscribers {
		s(h)
	}
}
