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

func TestK8sHealthChecker(t *testing.T) {
	typeMeta := metav1.TypeMeta{Kind: "Node", APIVersion: "v1"}

	node1Ready := corev1.Node{
		TypeMeta: typeMeta,
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
		TypeMeta: typeMeta,
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
		TypeMeta: typeMeta,
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

	clientNotReady := fake.NewSimpleClientset(
		&corev1.NodeList{
			Items: []corev1.Node{
				node1Ready,
				node2NotReady,
				node3Ready,
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
	)

	clientEmptyNodeList := fake.NewSimpleClientset(
		&corev1.NodeList{},
	)

	namespaces := []string{"kube-system"}
	clustername := "test"
	organizationid := uint(1)

	type fields struct {
		namespaces []string
	}
	type args struct {
		ctx            context.Context
		organizationID uint
		clusterName    string
		client         kubernetes.Interface
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
				ctx:            context.Background(),
				organizationID: organizationid,
				clusterName:    clustername,
				client:         clientNotReady,
			},
			wantErr: true,
		},
		{
			name: "all nodes are Ready",
			fields: fields{
				namespaces: namespaces,
			},
			args: args{
				ctx:            context.Background(),
				organizationID: organizationid,
				clusterName:    clustername,
				client:         clientReady,
			},
			wantErr: false,
		},
		{
			name: "empty nodelist",
			fields: fields{
				namespaces: namespaces,
			},
			args: args{
				ctx:            context.Background(),
				organizationID: organizationid,
				clusterName:    clustername,
				client:         clientEmptyNodeList,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := K8sHealthChecker{
				namespaces: tt.fields.namespaces,
			}
			if err := c.Check(tt.args.ctx, tt.args.organizationID, tt.args.clusterName, tt.args.client); (err != nil) != tt.wantErr {
				t.Errorf("K8sHealthChecker.Check() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
