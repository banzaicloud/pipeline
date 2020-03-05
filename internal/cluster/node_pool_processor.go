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
)

// NodePoolProcessors combines different node pool processors into one.
type NodePoolProcessors []NodePoolProcessor

// ProcessNew processes a new node pool descriptor.
func (p NodePoolProcessors) ProcessNew(
	ctx context.Context,
	cluster Cluster,
	rawNodePool NewRawNodePool,
) (NewRawNodePool, error) {
	for _, processor := range p {
		var err error

		rawNodePool, err = processor.ProcessNew(ctx, cluster, rawNodePool)
		if err != nil {
			return rawNodePool, err
		}
	}

	return rawNodePool, nil
}

type commonNodePoolProcessor struct {
	labelSource NodePoolLabelSource
}

// NewCommonNodePoolProcessor returns a new NodePoolProcessor
// that processes common node pool fields.
func NewCommonNodePoolProcessor(labelSource NodePoolLabelSource) NodePoolProcessor {
	return commonNodePoolProcessor{
		labelSource: labelSource,
	}
}

func (p commonNodePoolProcessor) ProcessNew(
	ctx context.Context,
	c Cluster,
	rawNodePool NewRawNodePool,
) (NewRawNodePool, error) {
	sourcedLabels, err := p.labelSource.GetLabels(ctx, c, rawNodePool)
	if err != nil {
		return rawNodePool, err
	}

	labels := rawNodePool.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	for key, value := range sourcedLabels {
		labels[key] = value
	}

	rawNodePool["labels"] = labels

	return rawNodePool, nil
}
