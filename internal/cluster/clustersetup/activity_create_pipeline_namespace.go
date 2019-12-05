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
	corev1 "k8s.io/api/core/v1"
	k8sapierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const CreatePipelineNamespaceActivityName = "create-pipeline-namespace"

type CreatePipelineNamespaceActivity struct {
	namespace string

	clientFactory ClientFactory
}

// NewCreatePipelineNamespaceActivity returns a new CreatePipelineNamespaceActivity.
func NewCreatePipelineNamespaceActivity(
	namespace string,
	clientFactory ClientFactory,
) CreatePipelineNamespaceActivity {
	return CreatePipelineNamespaceActivity{
		namespace:     namespace,
		clientFactory: clientFactory,
	}
}

type CreatePipelineNamespaceActivityInput struct {
	// Kubernetes cluster config secret ID.
	ConfigSecretID string
}

func (a CreatePipelineNamespaceActivity) Execute(ctx context.Context, input CreatePipelineNamespaceActivityInput) error {
	client, err := a.clientFactory.FromSecret(ctx, input.ConfigSecretID)
	if err != nil {
		return err
	}

	_, err = client.CoreV1().Namespaces().Create(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: a.namespace,
			Labels: map[string]string{
				"scan":  "noscan",
				"name":  a.namespace,
				"owner": "pipeline",
			},
		},
	})

	if err != nil && k8sapierrors.IsAlreadyExists(err) {
		return nil
	} else if err != nil {
		return errors.Wrap(err, "failed to create namespace")
	}

	return nil
}
