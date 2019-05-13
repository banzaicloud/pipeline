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

package pkeworkflow

import (
	"context"

	"github.com/banzaicloud/pipeline/pkg/cluster/pke"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/goph/emperror"
	"go.uber.org/cadence/activity"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const SetMasterTaintActivityName = "set-master-taint-activity"

// SetMasterTaintActivity sets the correct taints and labels for a single-node PKE cluster
type SetMasterTaintActivity struct {
	clusters Clusters
}

func NewSetMasterTaintActivity(clusters Clusters) *SetMasterTaintActivity {
	return &SetMasterTaintActivity{
		clusters: clusters,
	}
}

type SetMasterTaintActivityInput struct {
	ClusterID uint
}

func (a *SetMasterTaintActivity) Execute(ctx context.Context, input SetMasterTaintActivityInput) error {
	logger := activity.GetLogger(ctx).Sugar().With("clusterID", input.ClusterID)

	cluster, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return err
	}

	kubeConfig, err := cluster.GetK8sConfig()
	if err != nil {
		return err
	}

	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return err
	}

	nodes, err := client.CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: pke.TaintKeyMaster})
	if err != nil {
		return emperror.Wrap(err, "failed to list master nodes")
	}

	for _, node := range nodes.Items {
		logger := logger.With("node", node.Name)
		var taints []v1.Taint

		for _, taint := range node.Spec.Taints {
			if taint.Key != pke.TaintKeyMaster {
				taints = append(taints, taint)
			}
		}

		taints = append(taints, v1.Taint{
			Key:    pke.TaintKeyMaster,
			Effect: v1.TaintEffectPreferNoSchedule,
		})

		node.Spec.Taints = taints

		delete(node.ObjectMeta.Labels, pke.TaintKeyMaster)
		node.ObjectMeta.Labels[pke.NodeLabelKeyMasterWorker] = ""

		_, err = client.CoreV1().Nodes().Update(&node)
		if err != nil {
			return emperror.Wrapf(err, "failed to update node %q", node.ObjectMeta.Name)
		}

		logger.Info("tainted master node")
	}

	return nil
}
