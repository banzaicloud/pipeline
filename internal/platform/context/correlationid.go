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

import "context"

// CorrelationID returns a  correlation ID from a context (if any).
func CorrelationID(ctx context.Context) string {
	if cid, ok := ctx.Value(contextKeyCorrelationId).(string); ok {
		return cid
	}

	return ""
}

// WithCorrelationID returns a new context with the current correlation ID.
func WithCorrelationID(ctx context.Context, cid string) context.Context {
	return context.WithValue(ctx, contextKeyCorrelationId, cid)
}
