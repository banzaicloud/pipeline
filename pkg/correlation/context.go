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

	"go.opencensus.io/trace"
)

type contextKey string

func (c contextKey) String() string {
	return "trace context key " + string(c)
}

// ContextExtractor extracts values from a context.
type ContextExtractor struct{}

// Extract extracts values from a context.
func (*ContextExtractor) Extract(ctx context.Context) map[string]interface{} {
	fields := make(map[string]interface{})

	if correlationID, ok := ID(ctx); ok {
		fields["correlation_id"] = correlationID
	}

	if span := trace.FromContext(ctx); span != nil {
		spanCtx := span.SpanContext()
		fields["trace_id"] = spanCtx.TraceID.String()
		fields["span_id"] = spanCtx.SpanID.String()
	}

	return fields
}
