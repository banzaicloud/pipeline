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

package clustersetup

import (
	"context"

	"emperror.dev/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/pkg/k8sutil"
)

const LabelKubeSystemNamespaceActivityName = "label-kube-system-namespace"

type LabelKubeSystemNamespaceActivity struct {
	clientFactory ClientFactory
}

// NewLabelKubeSystemNamespaceActivity returns a new LabelKubeSystemNamespaceActivity.
func NewLabelKubeSystemNamespaceActivity(
	clientFactory ClientFactory,
) LabelKubeSystemNamespaceActivity {
	return LabelKubeSystemNamespaceActivity{
		clientFactory: clientFactory,
	}
}

type LabelKubeSystemNamespaceActivityInput struct {
	// Kubernetes cluster config secret ID.
	ConfigSecretID string
}

func (a LabelKubeSystemNamespaceActivity) Execute(ctx context.Context, input LabelKubeSystemNamespaceActivityInput) error {
	client, err := a.clientFactory.FromSecret(ctx, input.ConfigSecretID)
	if err != nil {
		return err
	}

	namespace, err := client.CoreV1().Namespaces().Get("kube-system", metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get namespace")
	}

	labels := map[string]string{"name": k8sutil.KubeSystemNamespace}

	if namespace.Labels == nil {
		namespace.Labels = make(map[string]string, len(labels))
	}

	for k, v := range labels {
		namespace.Labels[k] = v
	}

	_, err = client.CoreV1().Namespaces().Update(namespace)
	if err != nil {
		return errors.Wrap(err, "failed to update namespace")
	}

	return nil
}
