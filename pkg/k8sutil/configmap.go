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

package k8sutil

import (
	"encoding/json"
	"fmt"

	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

func GetConfigMapEntry(client kubernetes.Interface, namespace string, name string, key string) (value string, err error) {
	cm, err := client.CoreV1().ConfigMaps(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return "", emperror.Wrap(err, fmt.Sprintf("couldn't get configmap %s.%s", namespace, name))
	}
	if v, ok := cm.Data[key]; ok {
		return v, nil
	}
	return "", errors.Errorf("no entry with key %s", key)
}

func PatchConfigMapDataEntry(log logrus.FieldLogger, client kubernetes.Interface, namespace string, name string, key string, newValue string) error {
	log.Debugf("new value to add to configmap: %s=%s", key, newValue)
	patch := []patchOperation{{
		Op:    "add",
		Path:  "/data/" + key,
		Value: newValue,
	}}
	log.Debugf("patch: %v", patch)

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return emperror.Wrap(err, "failed to patch configmap")
	}

	_, err = client.CoreV1().ConfigMaps(namespace).Patch(name, types.JSONPatchType, patchBytes)
	if err != nil {
		return emperror.Wrap(err, "failed to patch configmap")
	}
	return nil
}
