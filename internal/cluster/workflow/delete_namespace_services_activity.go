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

package workflow

import (
	"context"

	"github.com/goph/emperror"
)

const DeleteNamespaceServicesActivityName = "delete-namespace-services"

type DeleteNamespaceServicesActivityInput struct {
	OrganizationID uint
	ClusterName    string
	K8sSecretID    string
	Namespace      string
}

type DeleteNamespaceServicesActivity struct {
	deleter         NamespaceServicesDeleter
	k8sConfigGetter K8sConfigGetter
}

type NamespaceServicesDeleter interface {
	Delete(organizationID uint, clusterName string, k8sConfig []byte, namespace string) error
}

func MakeDeleteNamespaceServicesActivity(deleter NamespaceServicesDeleter, k8sConfigGetter K8sConfigGetter) DeleteNamespaceServicesActivity {
	return DeleteNamespaceServicesActivity{
		deleter:         deleter,
		k8sConfigGetter: k8sConfigGetter,
	}
}

func (a DeleteNamespaceServicesActivity) Execute(ctx context.Context, input DeleteNamespaceServicesActivityInput) error {
	k8sConfig, err := a.k8sConfigGetter.Get(input.OrganizationID, input.K8sSecretID)
	if err != nil {
		return emperror.Wrap(err, "failed to get k8s config")
	}
	return emperror.Wrapf(a.deleter.Delete(input.OrganizationID, input.ClusterName, k8sConfig, input.Namespace), "failed to delete services in namespace %q", input.Namespace)
}
