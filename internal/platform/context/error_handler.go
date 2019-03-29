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

package context

import (
	"context"

	"github.com/goph/emperror"
)

// ErrorHandlerWithCorrelationID returns a new error handler with a correlation ID in its context.
func ErrorHandlerWithCorrelationID(ctx context.Context, errorHandler emperror.Handler) emperror.Handler {
	cid, ok := ctx.Value(contextKeyCorrelationId).(string)
	if !ok || cid == "" {
		return errorHandler
	}

	return emperror.HandlerWith(errorHandler, correlationIdField, cid)
}
