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

package logging

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type KubernetesService interface {
	// EnsureObject makes sure that a given Object is on the cluster and returns it.
	EnsureObject(ctx context.Context, clusterID uint, o runtime.Object) error

	// Update updates a given Object on the cluster and returns it.
	Update(ctx context.Context, clusterID uint, o runtime.Object) error

	// DeleteObject deletes an Object from a specific cluster.
	DeleteObject(ctx context.Context, clusterID uint, o runtime.Object) error

	// GetObject gets an Object from a specific cluster.
	GetObject(ctx context.Context, clusterID uint, objRef corev1.ObjectReference, obj runtime.Object) error

	// List lists Objects on specific cluster.
	List(ctx context.Context, clusterID uint, labels map[string]string, o runtime.Object) error
}
