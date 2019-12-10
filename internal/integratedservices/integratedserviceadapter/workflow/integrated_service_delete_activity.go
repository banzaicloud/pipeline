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

const IntegratedServiceDeleteActivityName = "integrated-service-delete"

type IntegratedServiceDeleteActivityInput struct {
	ClusterID             uint
	IntegratedServiceName string
}

type IntegratedServiceDeleteActivity struct {
	integratedServices integratedservices.IntegratedServiceRepository
}

func MakeIntegratedServiceDeleteActivity(integratedServices integratedservices.IntegratedServiceRepository) IntegratedServiceDeleteActivity {
	return IntegratedServiceDeleteActivity{
		integratedServices: integratedServices,
	}
}

func (a IntegratedServiceDeleteActivity) Execute(ctx context.Context, input IntegratedServiceDeleteActivityInput) error {
	return a.integratedServices.DeleteIntegratedService(ctx, input.ClusterID, input.IntegratedServiceName)
}
