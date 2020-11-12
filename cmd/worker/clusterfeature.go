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
	"go.uber.org/cadence/worker"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
	clusterfeatureworkflow "github.com/banzaicloud/pipeline/internal/integratedservices/integratedserviceadapter/workflow"
)

func registerClusterFeatureWorkflows(worker worker.Worker, featureOperatorRegistry integratedservices.IntegratedServiceOperatorRegistry, featureRepository integratedservices.IntegratedServiceRepository, isV2 bool) {
	workflowFunc := clusterfeatureworkflow.IntegratedServiceJobWorkflow
	if isV2 {
		workflowFunc = clusterfeatureworkflow.IntegratedServiceJobWorkflowV2
	}

	worker.RegisterWorkflowWithOptions(workflowFunc, workflow.RegisterOptions{Name: clusterfeatureworkflow.IntegratedServiceJobWorkflowName})

	{
		a := clusterfeatureworkflow.MakeIntegratedServicesApplyActivity(featureOperatorRegistry)
		worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: clusterfeatureworkflow.IntegratedServiceApplyActivityName})
	}

	{
		a := clusterfeatureworkflow.MakeIntegratedServiceDeleteActivity(featureRepository)
		worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: clusterfeatureworkflow.IntegratedServiceDeleteActivityName})
	}

	{
		a := clusterfeatureworkflow.MakeIntegratedServiceDeactivateActivity(featureOperatorRegistry)
		worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: clusterfeatureworkflow.IntegratedServiceDeactivateActivityName})
	}

	{
		a := clusterfeatureworkflow.MakeIntegratedServiceSetSpecActivity(featureRepository)
		worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: clusterfeatureworkflow.IntegratedServiceSetSpecActivityName})
	}

	{
		a := clusterfeatureworkflow.MakeIntegratedServiceSetStatusActivity(featureRepository)
		worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: clusterfeatureworkflow.IntegratedServiceSetStatusActivityName})
	}
}
