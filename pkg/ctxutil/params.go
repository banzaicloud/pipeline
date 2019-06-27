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

package ctxutil

import (
	"context"
)

// nolint: gochecknoglobals
var contextParams = contextKey("params")

// WithParams appends parameters to a context.
func WithParams(ctx context.Context, params map[string]string) context.Context {
	return context.WithValue(ctx, contextParams, params)
}

// Params fetches parameters from a context (if any).
func Params(ctx context.Context) (map[string]string, bool) {
	params, ok := ctx.Value(contextParams).(map[string]string)
	return params, ok
}
