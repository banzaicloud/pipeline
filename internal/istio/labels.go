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

package istio

import (
	"github.com/banzaicloud/pipeline/pkg/k8sutil"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

func LabelNamespaces(log logrus.FieldLogger, client kubernetes.Interface, namespaces []string) error {
	var nsLabels = map[string]string{
		"istio-injection": "enabled",
	}

	for _, ns := range namespaces {
		err := k8sutil.LabelNamespaceIgnoreNotFound(log, client, ns, nsLabels)
		if err != nil {
			return emperror.Wrap(err, "failed to label namespace")
		}
	}
	return nil
}
