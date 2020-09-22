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
	"github.com/sirupsen/logrus"
)

const K8sHealthCheckActivityName = "k8s-health-check"

type K8sHealthCheckActivityInput struct {
	OrganizationID uint
	ClusterName    string
	K8sSecretID    string
}

type K8sHealthCheckActivity struct {
	checker         K8sHealthChecker
	k8sConfigGetter K8sConfigGetter
	logger          logrus.FieldLogger
}

type K8sHealthChecker interface {
	Check(ctx context.Context, organizationID uint, clusterName string, k8sConfig []byte) error
}

func MakeK8sHealthCheckActivity(checker K8sHealthChecker, k8sConfigGetter K8sConfigGetter) K8sHealthCheckActivity {
	return K8sHealthCheckActivity{
		checker:         checker,
		k8sConfigGetter: k8sConfigGetter,
	}
}

func (a K8sHealthCheckActivity) Execute(ctx context.Context, input K8sHealthCheckActivityInput) error {
	k8sConfig, err := a.k8sConfigGetter.Get(input.OrganizationID, input.K8sSecretID)
	if err != nil {
		return errors.WrapIf(err, "failed to get k8s config")
	}

	err = a.checker.Check(ctx, input.OrganizationID, input.ClusterName, k8sConfig)
	return err
}
