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

package cluster

import (
	"context"
	"strconv"

	"emperror.dev/errors"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/global/globalcluster"
)

type NodePoolLabels struct {
	NodePoolName string
	Existing     bool
	InstanceType string            `json:"instanceType,omitempty"`
	SpotPrice    string            `json:"spotPrice,omitempty"`
	Preemptible  bool              `json:"preemptible,omitempty"`
	CustomLabels map[string]string `json:"labels,omitempty"`
}

// GetName returns the node pool name.
func (n NodePoolLabels) GetName() string {
	return n.NodePoolName
}

// GetInstanceType returns the node pool instance type.
func (n NodePoolLabels) GetInstanceType() string {
	return n.InstanceType
}

// IsOnDemand determines whether the machines in the node pool are on demand or spot/preemtible instances.
func (n NodePoolLabels) IsOnDemand() bool {
	if price, err := strconv.ParseFloat(n.SpotPrice, 64); err == nil {
		return price <= 0.0
	}

	return !n.Preemptible
}

// GetLabels returns labels that are/should be applied to every node in the pool.
func (n NodePoolLabels) GetLabels() map[string]string {
	return n.CustomLabels
}

// GetDesiredLabelsForCluster returns desired set of labels for each node pool name,
// adding reserved labels like: head node, ondemand labels + cloudinfo to user defined labels in specified nodePools map.
// All user labels are deleted in case Label map is empty in NodePoolLabels, however in case Label map is nil
// no labels are returned to avoid overriding already exisisting user specified labels.
func GetDesiredLabelsForCluster(ctx context.Context, cluster CommonCluster, nodePoolLabels []NodePoolLabels) (map[string]map[string]string, error) {
	desiredLabels := make(map[string]map[string]string)

	clusterStatus, err := cluster.GetStatus()
	if err != nil {
		return desiredLabels, errors.WrapIfWithDetails(err, "failed to get cluster status", "cluster", cluster.GetName())
	}

	for _, npLabels := range nodePoolLabels {
		labelsMap := getLabelsForNodePool(
			ctx,
			npLabels,
			clusterStatus.Cloud,
			clusterStatus.Distribution,
			clusterStatus.Region,
		)
		if len(labelsMap) > 0 {
			desiredLabels[npLabels.NodePoolName] = labelsMap
		}
	}
	return desiredLabels, nil
}

func getLabelsForNodePool(
	ctx context.Context,
	nodePool NodePoolLabels,
	cloud string,
	distribution string,
	region string,
) map[string]string {
	if nodePool.CustomLabels == nil && nodePool.Existing {
		return make(map[string]string)
	}

	labels, err := globalcluster.NodePoolLabelSource().GetLabels(
		ctx,
		cluster.Cluster{
			Cloud:        cloud,
			Distribution: distribution,
			Location:     region,
		},
		nodePool,
	)
	if err != nil {
		log.WithFields(logrus.Fields{
			"nodePool":     nodePool.NodePoolName,
			"instance":     nodePool.InstanceType,
			"cloud":        cloud,
			"distribution": distribution,
			"region":       region,
		}).Warn(errors.WithMessage(err, "failed to get labels for node pool"))
	}

	return labels
}
