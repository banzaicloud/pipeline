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

package main

import (
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
	clusterfeatureworkflow "github.com/banzaicloud/pipeline/internal/integratedservices/integratedserviceadapter/workflow"
)

func registerClusterFeatureWorkflows(featureOperatorRegistry integratedservices.IntegratedServiceOperatorRegistry, featureRepository integratedservices.IntegratedServiceRepository) {
	workflow.RegisterWithOptions(clusterfeatureworkflow.IntegratedServiceJobWorkflow, workflow.RegisterOptions{Name: clusterfeatureworkflow.IntegratedServiceJobWorkflowName})

	{
		a := clusterfeatureworkflow.MakeIntegratedServicesApplyActivity(featureOperatorRegistry)
		activity.RegisterWithOptions(a.Execute, activity.RegisterOptions{Name: clusterfeatureworkflow.IntegratedServiceApplyActivityName})
	}

	{
		a := clusterfeatureworkflow.MakeIntegratedServiceDeleteActivity(featureRepository)
		activity.RegisterWithOptions(a.Execute, activity.RegisterOptions{Name: clusterfeatureworkflow.IntegratedServiceDeleteActivityName})
	}

	{
		a := clusterfeatureworkflow.MakeIntegratedServiceDeactivateActivity(featureOperatorRegistry)
		activity.RegisterWithOptions(a.Execute, activity.RegisterOptions{Name: clusterfeatureworkflow.IntegratedServiceDeactivateActivityName})
	}

	{
		a := clusterfeatureworkflow.MakeIntegratedServiceSetSpecActivity(featureRepository)
		activity.RegisterWithOptions(a.Execute, activity.RegisterOptions{Name: clusterfeatureworkflow.IntegratedServiceSetSpecActivityName})
	}

	{
		a := clusterfeatureworkflow.MakeIntegratedServiceSetStatusActivity(featureRepository)
		activity.RegisterWithOptions(a.Execute, activity.RegisterOptions{Name: clusterfeatureworkflow.IntegratedServiceSetStatusActivityName})
	}
}
