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

package npls

import (
	"emperror.dev/errors"
	"github.com/mitchellh/mapstructure"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// nolint: gochecknoglobals
var (
	labelsetGVR = schema.GroupVersionResource{
		Group:    "labels.banzaicloud.io",
		Version:  "v1alpha1",
		Resource: "nodepoollabelsets",
	}
)

// Manager synchronizes nodepool labels.
type Manager struct {
	client    dynamic.Interface
	namespace string
}

// NewManager returns a new Manager.
func NewManager(client dynamic.Interface, namespace string) Manager {
	return Manager{
		client:    client,
		namespace: namespace,
	}
}

func (m Manager) GetAll() (map[string]map[string]string, error) {
	client := m.client.Resource(labelsetGVR).Namespace(m.namespace)

	list, err := client.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	labelSets := make(map[string]map[string]string, len(list.Items))

	for _, item := range list.Items {
		var labelSet struct {
			Spec struct {
				Labels map[string]string `mapstructure:"labels"`
			} `mapstructure:"spec"`
		}

		err := mapstructure.Decode(item.Object, &labelSet)
		if err != nil {
			return nil, err
		}

		labelSets[item.GetName()] = labelSet.Spec.Labels
	}

	return labelSets, nil
}

func (m Manager) Sync(labels map[string]map[string]string) error {
	client := m.client.Resource(labelsetGVR).Namespace(m.namespace)

	errs := make([]error, 0, len(labels))

	for poolName, labelSet := range labels {
		if len(labelSet) == 0 {
			err := client.Delete(poolName, nil)
			if k8serrors.IsNotFound(err) {
				continue
			}
			if err != nil {
				errs = append(errs, err)
			}
			continue
		}

		currentLabelSetObj, err := client.Get(poolName, metav1.GetOptions{})
		if err != nil && k8serrors.IsNotFound(errors.Cause(err)) {
			labelSetObj := unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":       "NodePoolLabelSet",
					"apiVersion": labelsetGVR.GroupVersion().String(),
					"metadata": map[string]interface{}{
						"name": poolName,
					},
					"spec": map[string]interface{}{
						"labels": labelSet,
					},
				},
			}

			_, err := client.Create(&labelSetObj, metav1.CreateOptions{})
			if err != nil {
				errs = append(errs, err)
			}

			continue
		} else if err != nil {
			errs = append(errs, err)

			continue
		}

		labelSetObj := unstructured.Unstructured{
			Object: map[string]interface{}{
				"kind":       "NodePoolLabelSet",
				"apiVersion": labelsetGVR.GroupVersion().String(),
				"metadata": map[string]interface{}{
					"name":            poolName,
					"resourceVersion": currentLabelSetObj.GetResourceVersion(),
				},
				"spec": map[string]interface{}{
					"labels": labelSet,
				},
			},
		}

		_, err = client.Update(&labelSetObj, metav1.UpdateOptions{})
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Combine(errs...)
}
