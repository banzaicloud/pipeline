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

package istiofeature

import (
	"context"

	"emperror.dev/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (m *MeshReconciler) applyResource(client runtimeclient.Client, desired runtime.Object) error {
	key, err := runtimeclient.ObjectKeyFromObject(desired)
	if err != nil {
		return errors.WrapIf(err, "could not get object key")
	}

	current := desired.DeepCopyObject()
	err = client.Get(context.Background(), key, current)
	if err != nil && !k8serrors.IsNotFound(err) {
		return errors.WrapIf(err, "could not get resource")
	}

	if k8serrors.IsNotFound(err) {
		err = client.Create(context.Background(), desired)
		if err != nil {
			return errors.WrapIf(err, "could not create resource")
		}
	} else if err == nil {
		metaAccessor := meta.NewAccessor()
		currentResourceVersion, err := metaAccessor.ResourceVersion(current)
		if err != nil {
			return errors.WrapIf(err, "could not get resource version for update")
		}

		err = metaAccessor.SetResourceVersion(desired, currentResourceVersion)
		if err != nil {
			return errors.WrapIf(err, "could not set resource version for update")
		}

		m.prepareResourceForUpdate(current, desired)

		err = client.Update(context.Background(), desired)
		if err != nil {
			return errors.WrapIf(err, "could not update resource")
		}
	}

	return nil
}

func (m *MeshReconciler) deleteResource(client runtimeclient.Client, obj runtime.Object) error {
	key, err := runtimeclient.ObjectKeyFromObject(obj)
	if err != nil {
		return errors.WrapIf(err, "could not get object key")
	}

	err = client.Get(context.Background(), key, obj)
	if err != nil && !k8serrors.IsNotFound(err) {
		return errors.WrapIf(err, "could not get resource")
	}

	err = client.Delete(context.Background(), obj)
	if err != nil && !k8serrors.IsNotFound(err) {
		return errors.WrapIf(err, "could not remove resource")
	}

	return nil
}

func (m *MeshReconciler) prepareResourceForUpdate(current, desired runtime.Object) {
	switch desired.(type) {
	case *corev1.Service:
		svc := desired.(*corev1.Service)
		svc.Spec.ClusterIP = current.(*corev1.Service).Spec.ClusterIP
	}
}
