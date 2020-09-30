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

const HealthCheckActivityName = "k8s-health-check"

type HealthCheckActivityInput struct {
	SecretID string
}

type HealthCheckActivity struct {
	checker       HealthChecker
	clientFactory ClientFactory
}

// +testify:mock:testOnly=true

// HealthChecker returns the result of the healthcheck
type HealthChecker interface {
	Check(ctx context.Context, client kubernetes.Interface) error
}

// NewHealthCheckActivity returns a new HealthCheckActivity.
func NewHealthCheckActivity(
	checker HealthChecker, clientFactory ClientFactory,
) HealthCheckActivity {
	return HealthCheckActivity{
		checker:       checker,
		clientFactory: clientFactory,
	}
}

func (a HealthCheckActivity) Execute(ctx context.Context, input HealthCheckActivityInput) error {
	client, err := a.clientFactory.FromSecret(ctx, input.SecretID)
	if err != nil {
		return err
	}

	return a.checker.Check(ctx, client)
}
