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
	"time"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
)

const IntegratedServiceApplyActivityName = "integrated-service-apply"

type IntegratedServiceApplyActivityInput struct {
	ClusterID             uint
	IntegratedServiceName string
	IntegratedServiceSpec integratedservices.IntegratedServiceSpec
	RetryInterval         time.Duration
}

type IntegratedServiceApplyActivity struct {
	integratedServices integratedservices.IntegratedServiceOperatorRegistry
}

func MakeIntegratedServicesApplyActivity(integratedServices integratedservices.IntegratedServiceOperatorRegistry) IntegratedServiceApplyActivity {
	return IntegratedServiceApplyActivity{
		integratedServices: integratedServices,
	}
}

func (a IntegratedServiceApplyActivity) Execute(ctx context.Context, input IntegratedServiceApplyActivityInput) error {
	f, err := a.integratedServices.GetIntegratedServiceOperator(input.IntegratedServiceName)
	if err != nil {
		return err
	}

	heartbeat := startHeartbeat(ctx, 10*time.Second)
	defer heartbeat.Stop()

	for {
		if err := f.Apply(ctx, input.ClusterID, input.IntegratedServiceSpec); err != nil {
			if shouldRetry(err) {
				if err := wait(ctx, input.RetryInterval); err != nil {
					return err
				}

				continue
			}

			return err
		}

		return nil
	}
}
