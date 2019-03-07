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
	"encoding/json"

	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/goph/emperror"
	"go.uber.org/cadence/activity"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const SetMasterTaintActivityName = "set-master-taint-activity"
const masterKey = "node-role.kubernetes.io/master"

type SetMasterTaintActivity struct {
	clusters Clusters
}

func NewSetMasterTaintActivity(clusters Clusters) *SetMasterTaintActivity {
	return &SetMasterTaintActivity{
		clusters: clusters,
	}
}

type SetMasterTaintActivityInput struct {
	ClusterID   uint
	Schedulable bool
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

	selector := "node-role.kubernetes.io/master"

	effect := v1.TaintEffectNoSchedule
	if input.Schedulable {
		effect = v1.TaintEffectPreferNoSchedule
	}

	nodes, err := client.CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return emperror.Wrap(err, "failed to list master nodes")
	}

	for _, node := range nodes.Items {
		logger := logger.With("node", node.Name)
		taints := []v1.Taint{}

		for _, taint := range node.Spec.Taints {
			if taint.Key != masterKey {
				taints = append(taints, taint)
			}
		}

		taints = append(taints, v1.Taint{
			Key:    masterKey,
			Effect: effect,
		})

		patch := v1.Node{Spec: v1.NodeSpec{Taints: taints}}
		patchData, err := json.Marshal(patch)
		if err != nil {
			return emperror.Wrap(err, "failed to marshal node patch")
		}

		_, err = client.CoreV1().Nodes().Patch(node.Name, types.StrategicMergePatchType, patchData)
		if err != nil {
			return err
		}
		logger.Info("tainted master node")
	}

	return nil
}
