// Copyright © 2020 Banzai Cloud
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

package appkit

import (
	"context"

	"github.com/sagikazarmark/kitx/correlation"
	kitxendpoint "github.com/sagikazarmark/kitx/endpoint"
	"go.opencensus.io/trace"
)

// ContextExtractor extracts fields from a context.
func ContextExtractor(ctx context.Context) map[string]interface{} {
	fields := make(map[string]interface{})

	if correlationID, ok := correlation.FromContext(ctx); ok {
		fields["correlationId"] = correlationID
	}

	if operationName, ok := kitxendpoint.OperationName(ctx); ok {
		fields["operationName"] = operationName
	}

	if span := trace.FromContext(ctx); span != nil {
		spanCtx := span.SpanContext()

		fields["traceId"] = spanCtx.TraceID.String()
		fields["spanId"] = spanCtx.SpanID.String()
	}

	return fields
}
