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

package workflow

import (
	"context"

	"k8s.io/client-go/dynamic"
)

// DynamicClientFactory returns a dynamic Kubernetes client.
type DynamicClientFactory interface {
	// FromClusterID creates a dynamic Kubernetes client for a cluster from a cluster ID.
	FromClusterID(ctx context.Context, clusterID uint) (dynamic.Interface, error)
}
