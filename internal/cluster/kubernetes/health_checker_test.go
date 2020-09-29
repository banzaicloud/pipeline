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

package kubernetes

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCheckNodeStatus(t *testing.T) {
	type args struct {
		node corev1.Node
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "node is ready",
			args: args{
				node: corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node1",
					},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{
								Type:   corev1.NodeReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.NodeMemoryPressure,
								Status: corev1.ConditionFalse,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "node is not ready",
			args: args{
				node: corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node1",
					},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{
								Type:   corev1.NodeReady,
								Status: corev1.ConditionFalse,
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "node is unknown state",
			args: args{
				node: corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node1",
					},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{
								Type:   corev1.NodeReady,
								Status: corev1.ConditionUnknown,
							},
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkNodeStatus(tt.args.node); (err != nil) != tt.wantErr {
				t.Errorf("checkNodeStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHealthChecker(t *testing.T) {
	typeMetaNode := metav1.TypeMeta{Kind: "Node", APIVersion: "v1"}
	typeMetaPod := metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"}

	namespaces := []string{"kube-system"}

	node1Ready := corev1.Node{
		TypeMeta: typeMetaNode,
		ObjectMeta: metav1.ObjectMeta{
			Name: "node1",
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	node2NotReady := corev1.Node{
		TypeMeta: typeMetaNode,
		ObjectMeta: metav1.ObjectMeta{
			Name: "node2",
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionFalse,
				},
			},
		},
	}

	node3Ready := corev1.Node{
		TypeMeta: typeMetaNode,
		ObjectMeta: metav1.ObjectMeta{
			Name: "node3",
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	pod1Ready := corev1.Pod{
		TypeMeta: typeMetaPod,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: namespaces[0],
		},
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
			Phase: corev1.PodRunning,
		},
	}

	pod2NotReady := corev1.Pod{
		TypeMeta: typeMetaPod,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod2",
			Namespace: namespaces[0],
		},
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionFalse,
				},
			},
			Phase: corev1.PodRunning,
		},
	}

	pod3NotRunning := corev1.Pod{
		TypeMeta: typeMetaPod,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod3",
			Namespace: namespaces[0],
		},
		Status: corev1.PodStatus{},
	}

	pod4Completed := corev1.Pod{
		TypeMeta: typeMetaPod,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod4",
			Namespace: namespaces[0],
		},
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionFalse,
				},
			},
			Phase: corev1.PodSucceeded,
		},
	}

	clientNodeNotReady := fake.NewSimpleClientset(
		&corev1.NodeList{
			Items: []corev1.Node{
				node1Ready,
				node2NotReady,
				node3Ready,
			},
		},
		&corev1.PodList{
			Items: []corev1.Pod{
				pod1Ready,
			},
		},
	)

	clientReady := fake.NewSimpleClientset(
		&corev1.NodeList{
			Items: []corev1.Node{
				node1Ready,
				node3Ready,
			},
		},
		&corev1.PodList{
			Items: []corev1.Pod{
				pod1Ready,
				pod4Completed,
			},
		},
	)

	clientPodNotReady := fake.NewSimpleClientset(
		&corev1.NodeList{
			Items: []corev1.Node{
				node1Ready,
				node3Ready,
			},
		},
		&corev1.PodList{
			Items: []corev1.Pod{
				pod2NotReady,
			},
		},
	)

	clientPodNotRunning := fake.NewSimpleClientset(
		&corev1.NodeList{
			Items: []corev1.Node{
				node1Ready,
				node3Ready,
			},
		},
		&corev1.PodList{
			Items: []corev1.Pod{
				pod3NotRunning,
			},
		},
	)

	clientEmptyNodeList := fake.NewSimpleClientset(
		&corev1.NodeList{},
	)

	clientEmptyPodList := fake.NewSimpleClientset(
		&corev1.NodeList{
			Items: []corev1.Node{
				node1Ready,
				node3Ready,
			},
		},
		&corev1.PodList{},
	)

	type fields struct {
		namespaces []string
	}
	type args struct {
		ctx    context.Context
		client kubernetes.Interface
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "not all nodes are Ready",
			fields: fields{
				namespaces: namespaces,
			},
			args: args{
				ctx:    context.Background(),
				client: clientNodeNotReady,
			},
			wantErr: true,
		},
		{
			name: "all nodes and pods are Ready",
			fields: fields{
				namespaces: namespaces,
			},
			args: args{
				ctx:    context.Background(),
				client: clientReady,
			},
			wantErr: false,
		},
		{
			name: "not all pods are Ready",
			fields: fields{
				namespaces: namespaces,
			},
			args: args{
				ctx:    context.Background(),
				client: clientPodNotReady,
			},
			wantErr: true,
		},
		{
			name: "not all pods are Running",
			fields: fields{
				namespaces: namespaces,
			},
			args: args{
				ctx:    context.Background(),
				client: clientPodNotRunning,
			},
			wantErr: true,
		},
		{
			name: "empty nodelist",
			fields: fields{
				namespaces: namespaces,
			},
			args: args{
				ctx:    context.Background(),
				client: clientEmptyNodeList,
			},
			wantErr: true,
		},
		{
			name: "empty podlist",
			fields: fields{
				namespaces: namespaces,
			},
			args: args{
				ctx:    context.Background(),
				client: clientEmptyPodList,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := HealthChecker{
				namespaces: tt.fields.namespaces,
			}
			if err := c.Check(tt.args.ctx, tt.args.client); (err != nil) != tt.wantErr {
				t.Errorf("K8sHealthChecker.Check() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
