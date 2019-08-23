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

package log

import (
	"emperror.dev/errors"
	logurhandler "emperror.dev/handler/logur"
	"k8s.io/apimachinery/pkg/util/runtime"
	"logur.dev/logur"
)

// SetK8sLogger overrides the default klog instance.
// By default klog tries to write into the filesystem, which doesn't work with scratch containers (there is no /tmp dir),
// so we override it entirely.
// See https://github.com/kubernetes/apimachinery/blob/052f7ea/pkg/util/runtime/runtime.go#L78
func SetK8sLogger(logger logur.Logger) {
	// TODO: use main error handler?
	handler := logurhandler.WithStackInfo(logurhandler.New(logger))

	runtime.ErrorHandlers[0] = func(e error) {
		e = errors.WithStackDepthIf(e, 2)

		handler.Handle(e)
	}
}
