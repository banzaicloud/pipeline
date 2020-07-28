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

package errorhandler

import (
	"emperror.dev/emperror"
	logurhandler "emperror.dev/handler/logur"
	"logur.dev/logur"
)

// New returns a new error handler.
func New(logger logur.Logger) (emperror.ErrorHandlers, error) {
	logHandler := logurhandler.WithStackInfo(logurhandler.New(logger))
	handlers := emperror.ErrorHandlers{logHandler}

	return handlers, nil
}
