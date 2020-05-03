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

package cluster

import (
	"context"
	"strconv"

	"emperror.dev/errors"
)

// Common node pool label constants
const (
	NodePoolNameLabelKey    = "nodepool.banzaicloud.io/name"
	NodePoolVersionLabelKey = "nodepool.banzaicloud.io/version"

	nodeOnDemandLabelKey = "node.banzaicloud.io/ondemand"
)

// +testify:mock:testOnly=true

// NodePoolLabelSource returns a set of labels that should be applied to every node in the pool.
type NodePoolLabelSource interface {
	// GetLabels returns a set of labels that should be applied to every node in the pool.
	GetLabels(ctx context.Context, cluster Cluster, nodePool NodePool) (map[string]string, error)
}

// NodePoolLabelSources combines different node pool label sources into one.
// In case of conflicting labels, the last one is applied (in the order of sources).
type NodePoolLabelSources []NodePoolLabelSource

// GetLabels returns a set of labels that should be applied to every node in the pool.
func (s NodePoolLabelSources) GetLabels(ctx context.Context, cluster Cluster, nodePool NodePool) (map[string]string, error) {
	var errs []error

	labels := make(map[string]string)

	for _, source := range s {
		l, err := source.GetLabels(ctx, cluster, nodePool)
		if err != nil {
			errs = append(errs, err)

			continue
		}

		// Merge with existing labels
		for key, value := range l {
			labels[key] = value
		}
	}

	return labels, errors.Combine(errs...)
}

type commonNodePoolLabelSource struct{}

// NewCommonNodePoolLabelSource returns a new NodePoolLabelSource
// that returns common labels for a node pool.
func NewCommonNodePoolLabelSource() NodePoolLabelSource {
	return commonNodePoolLabelSource{}
}

func (s commonNodePoolLabelSource) GetLabels(_ context.Context, _ Cluster, nodePool NodePool) (map[string]string, error) {
	return map[string]string{
		NodePoolNameLabelKey: nodePool.GetName(),
		nodeOnDemandLabelKey: strconv.FormatBool(nodePool.IsOnDemand()),
	}, nil
}

type filterValidNodePoolLabelSource struct {
	labelValidator LabelValidator
}

// NewFilterValidNodePoolLabelSource returns a new NodePoolLabelSource
// that validates existing labels and filters invalid ones.
func NewFilterValidNodePoolLabelSource(labelValidator LabelValidator) NodePoolLabelSource {
	return filterValidNodePoolLabelSource{
		labelValidator: labelValidator,
	}
}

func (s filterValidNodePoolLabelSource) GetLabels(_ context.Context, _ Cluster, nodePool NodePool) (map[string]string, error) {
	labels := make(map[string]string)

	for key, value := range nodePool.GetLabels() {
		if s.labelValidator.ValidateKey(key) != nil || s.labelValidator.ValidateValue(value) != nil {
			continue
		}

		labels[key] = value
	}

	return labels, nil
}
