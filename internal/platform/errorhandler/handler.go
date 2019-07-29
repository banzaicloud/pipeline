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
	"emperror.dev/errors"
)

// Handlers collects a number of error handlers into a single one.
// TODO(sagikazarmark): Use emperror.dev/emperror.Handlers instead.
type Handlers []emperror.Handler

func (h Handlers) Handle(err error) {
	for _, handler := range h {
		handler.Handle(err)
	}
}

// Close calls Close on the underlying handlers (if there is any closable handler).
func (h Handlers) Close() error {
	if len(h) < 1 {
		return nil
	}

	errs := make([]error, len(h))

	for i, handler := range h {
		if closer, ok := handler.(interface{ Close() error }); ok {
			errs[i] = closer.Close()
		}
	}

	return errors.Combine(errs...)
}
