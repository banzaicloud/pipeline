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

	"k8s.io/client-go/kubernetes"
)

const K8sHealthCheckActivityName = "k8s-health-check"

type K8sHealthCheckActivityInput struct {
	K8sSecretID string
}

type K8sHealthCheckActivity struct {
	checker       K8sHealthChecker
	clientFactory ClientFactory
}

// +testify:mock:testOnly=true

// K8sHealthChecker returns the result of the healthcheck
type K8sHealthChecker interface {
	Check(ctx context.Context, client kubernetes.Interface) error
}

// MakeK8sHealthCheckActivity returns a new K8sHealthCheckActivity.
func MakeK8sHealthCheckActivity(
	checker K8sHealthChecker, clientFactory ClientFactory,
) K8sHealthCheckActivity {
	return K8sHealthCheckActivity{
		checker:       checker,
		clientFactory: clientFactory,
	}
}

func (a K8sHealthCheckActivity) Execute(ctx context.Context, input K8sHealthCheckActivityInput) error {
	client, err := a.clientFactory.FromSecret(ctx, input.K8sSecretID)
	if err != nil {
		return err
	}

	err = a.checker.Check(ctx, client)
	return err
}
