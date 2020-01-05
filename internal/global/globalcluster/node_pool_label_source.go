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

package globalcluster

import (
	"context"
	"sync"

	"github.com/banzaicloud/pipeline/internal/cluster"
)

// nolint: gochecknoglobals
var nplSource LabelSource

// nolint: gochecknoglobals
var nplSourceMu sync.Mutex

type LabelSource interface {
	GetLabels(ctx context.Context, cluster cluster.Cluster, nodePool cluster.NodePool) (map[string]string, error)
}

// NodePoolLabelSource returns an initialized cloudinfo client.
func NodePoolLabelSource() LabelSource {
	nplSourceMu.Lock()
	defer nplSourceMu.Unlock()

	return nplSource
}

// SetNodePoolLabelSource configures a cloudinfo client.
func SetNodePoolLabelSource(s LabelSource) {
	nplSourceMu.Lock()
	defer nplSourceMu.Unlock()

	nplSource = s
}
