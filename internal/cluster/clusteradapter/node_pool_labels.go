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

package clusteradapter

import (
	"context"
	"regexp"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/pkg/cloudinfo"
)

// Constants for labeling cluster nodes
const (
	labelFormatRegexp      = "[^-A-Za-z0-9_.]"
	nodePoolLabelKeyPrefix = "node.banzaicloud.io/"
)

// nolint: gochecknoglobals
var labelFormatRe = regexp.MustCompile(labelFormatRegexp)

type cloudinfoNodePoolLabelSource struct {
	client *cloudinfo.Client
}

// NewCloudinfoNodePoolLabelSource returns a new cluster.NodePoolLabelSource
// that gets default node pool labels from Cloudinfo.
func NewCloudinfoNodePoolLabelSource(client *cloudinfo.Client) cluster.NodePoolLabelSource {
	return cloudinfoNodePoolLabelSource{
		client: client,
	}
}

func (s cloudinfoNodePoolLabelSource) GetLabels(
	ctx context.Context,
	c cluster.Cluster,
	nodePool cluster.NodePool,
) (map[string]string, error) {
	details, err := s.client.GetProductDetails(
		ctx,
		c.Cloud,
		c.Distribution,
		c.Location,
		nodePool.GetInstanceType(),
	)
	if err != nil {
		return nil, err
	}

	labels := make(map[string]string, len(details.Attributes))

	for key, value := range details.Attributes {
		labels[nodePoolLabelKeyPrefix+labelFormatRe.ReplaceAllString(key, "_")] = labelFormatRe.ReplaceAllString(value, "_")
	}

	return labels, nil
}
