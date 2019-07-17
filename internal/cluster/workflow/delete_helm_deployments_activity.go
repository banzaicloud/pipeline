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

	"emperror.dev/emperror"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/helm"
)

const DeleteHelmDeploymentsActivityName = "delete-helm-deployments"

type DeleteHelmDeploymentsActivityInput struct {
	OrganizationID uint
	ClusterName    string
	K8sSecretID    string
}

type DeleteHelmDeploymentsActivity struct {
	k8sConfigGetter K8sConfigGetter
	logger          logrus.FieldLogger
}

func MakeDeleteHelmDeploymentsActivity(k8sConfigGetter K8sConfigGetter, logger logrus.FieldLogger) DeleteHelmDeploymentsActivity {
	return DeleteHelmDeploymentsActivity{
		k8sConfigGetter: k8sConfigGetter,
		logger:          logger,
	}
}

func (a DeleteHelmDeploymentsActivity) Execute(ctx context.Context, input DeleteHelmDeploymentsActivityInput) error {
	logger := a.logger.WithField("organizationID", input.OrganizationID).WithField("clusterName", input.ClusterName)
	k8sConfig, err := a.k8sConfigGetter.Get(input.OrganizationID, input.K8sSecretID)
	if err != nil {
		return emperror.Wrap(err, "failed to get k8s config")
	}
	return emperror.Wrap(helm.DeleteAllDeployment(logger, k8sConfig), "failed to delete all Helm deployments")
}
