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

package correlation

import (
	"context"
)

// nolint: gochecknoglobals
var correlationID = contextKey("correlation-id")

// WithID returns a new context annotated with a correlation ID.
func WithID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationID, id)
}

// ID is awesome.
func ID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(correlationID).(string)

	return id, ok
}
