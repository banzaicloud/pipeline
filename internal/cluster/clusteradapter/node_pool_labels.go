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

	"github.com/banzaicloud/pipeline/pkg/cloudinfo"
)

// Constants for labeling cluster nodes
const (
	labelFormatRegexp      = "[^-A-Za-z0-9_.]"
	nodePoolLabelKeyPrefix = "node.banzaicloud.io/"
)

// nolint: gochecknoglobals
var labelFormatRe = regexp.MustCompile(labelFormatRegexp)

// CloudinfoNodePoolLabelSource gets default node pool labels from Cloudinfo.
type CloudinfoNodePoolLabelSource struct {
	client *cloudinfo.Client
}

// NewCloudinfoNodePoolLabelSource returns a new CloudinfoNodePoolLabelSource.
func NewCloudinfoNodePoolLabelSource(client *cloudinfo.Client) CloudinfoNodePoolLabelSource {
	return CloudinfoNodePoolLabelSource{
		client: client,
	}
}

// GetLabels returns node pool labels.
func (s CloudinfoNodePoolLabelSource) GetLabels(
	ctx context.Context,
	cloud string,
	distribution string,
	region string,
	instanceType string,
) (map[string]string, error) {
	details, err := s.client.GetProductDetails(
		ctx,
		cloud,
		distribution,
		region,
		instanceType,
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
