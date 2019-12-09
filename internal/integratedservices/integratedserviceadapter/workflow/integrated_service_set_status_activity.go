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

	"github.com/banzaicloud/pipeline/internal/integratedservices"
)

const IntegratedServiceSetStatusActivityName = "integrated-service-set-status"

type IntegratedServiceSetStatusActivityInput struct {
	ClusterID             uint
	IntegratedServiceName string
	Status                string
}

type IntegratedServiceSetStatusActivity struct {
	integratedServices integratedservices.IntegratedServiceRepository
}

func MakeIntegratedServiceSetStatusActivity(integratedServices integratedservices.IntegratedServiceRepository) IntegratedServiceSetStatusActivity {
	return IntegratedServiceSetStatusActivity{
		integratedServices: integratedServices,
	}
}

func (a IntegratedServiceSetStatusActivity) Execute(ctx context.Context, input IntegratedServiceSetStatusActivityInput) error {
	return a.integratedServices.UpdateIntegratedServiceStatus(ctx, input.ClusterID, input.IntegratedServiceName, input.Status)
}
