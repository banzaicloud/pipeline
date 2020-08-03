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

package workflow

import (
	"context"

	"emperror.dev/errors"
	"go.uber.org/cadence/activity"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

const DeleteK8sNodeActivityName = "delete-k8s-node"

type DeleteK8sNodeActivityInput struct {
	OrganizationID uint
	ClusterName    string
	K8sSecretID    string
	Name           string
}

type K8sConfigGetter interface {
	Get(organizationID uint, k8sSecretID string) ([]byte, error)
}

type DeleteK8sNodeActivity struct {
	k8sConfigGetter K8sConfigGetter
}

func MakeDeleteK8sNodeActivity(k8sConfigGetter K8sConfigGetter) DeleteK8sNodeActivity {
	return DeleteK8sNodeActivity{
		k8sConfigGetter: k8sConfigGetter,
	}
}

func (a DeleteK8sNodeActivity) Execute(ctx context.Context, input DeleteK8sNodeActivityInput) error {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"node", input.Name,
	)

	k8sConfig, err := a.k8sConfigGetter.Get(input.OrganizationID, input.K8sSecretID)
	if err != nil {
		return errors.WrapIf(err, "failed to get k8s config")
	}

	client, err := k8sclient.NewClientFromKubeConfig(k8sConfig)
	if err != nil {
		return err
	}

	_, err = client.CoreV1().Nodes().Get(ctx, input.Name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		logger.Info("node already deleted from Kubernetes")
		return nil
	}

	err = client.CoreV1().Nodes().Delete(ctx, input.Name, metav1.DeleteOptions{})
	return errors.WrapIff(err, "failed to delete node %q", input.Name)
}
