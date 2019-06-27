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

// TODO: move this to an internal pkg?

// nolint: gochecknoglobals
var contextClusterID = contextKey("cluster-id")

// ClusterID fetches cluster ID from a context (if any).
func ClusterID(ctx context.Context) (uint, bool) {
	clusterID, ok := ctx.Value(contextClusterID).(uint)
	return clusterID, ok
}

// WithClusterID appends a cluster ID to a context.
func WithClusterID(ctx context.Context, clusterID uint) context.Context {
	return context.WithValue(ctx, contextClusterID, clusterID)
}
