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

	"github.com/banzaicloud/pipeline/helm"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
)

const DeleteHelmDeploymentsActivityName = "delete-helm-deployments"

type DeleteHelmDeploymentsActivityInput struct {
	OrganizationID uint
	ClusterName    string
	K8sConfig      []byte
}

type DeleteHelmDeploymentsActivity struct {
	logger logrus.FieldLogger
}

func MakeDeleteHelmDeploymentsActivity(logger logrus.FieldLogger) DeleteHelmDeploymentsActivity {
	return DeleteHelmDeploymentsActivity{
		logger: logger,
	}
}

func (a DeleteHelmDeploymentsActivity) Execute(ctx context.Context, input DeleteHelmDeploymentsActivityInput) error {
	logger := a.logger.WithField("organizationID", input.OrganizationID).WithField("clusterName", input.ClusterName)
	return emperror.Wrap(helm.DeleteAllDeployment(logger, input.K8sConfig), "failed to delete all Helm deployments")
}
