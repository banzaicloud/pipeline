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
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
