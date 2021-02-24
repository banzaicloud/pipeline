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

func registerClusterFeatureWorkflows(worker worker.Worker, featureOperatorRegistry integratedservices.IntegratedServiceOperatorRegistry, featureRepository integratedservices.IntegratedServiceRepository, workflowName string, isV2 bool) {
	if isV2 {
		worker.RegisterWorkflowWithOptions(clusterfeatureworkflow.IntegratedServiceJobWorkflowV2, workflow.RegisterOptions{Name: workflowName})
	} else {
		worker.RegisterWorkflowWithOptions(clusterfeatureworkflow.IntegratedServiceJobWorkflow, workflow.RegisterOptions{Name: workflowName})
	}

	{
		activityName := clusterfeatureworkflow.GetActivityName(clusterfeatureworkflow.IntegratedServiceApplyActivityName, isV2)
		a := clusterfeatureworkflow.MakeIntegratedServicesApplyActivity(featureOperatorRegistry)
		worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: activityName})
	}

	{
		if !isV2 {
			activityName := clusterfeatureworkflow.GetActivityName(clusterfeatureworkflow.IntegratedServiceDeleteActivityName, isV2)
			a := clusterfeatureworkflow.MakeIntegratedServiceDeleteActivity(featureRepository)
			worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: activityName})
		}
	}

	{
		activityName := clusterfeatureworkflow.GetActivityName(clusterfeatureworkflow.IntegratedServiceDeactivateActivityName, isV2)
		a := clusterfeatureworkflow.MakeIntegratedServiceDeactivateActivity(featureOperatorRegistry)
		worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: activityName})
	}

	{
		if !isV2 {
			// this activity is not used
			activityName := clusterfeatureworkflow.GetActivityName(clusterfeatureworkflow.IntegratedServiceSetSpecActivityName, isV2)
			a := clusterfeatureworkflow.MakeIntegratedServiceSetSpecActivity(featureRepository)
			worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: activityName})
		}
	}

	{
		if !isV2 {
			activityName := clusterfeatureworkflow.GetActivityName(clusterfeatureworkflow.IntegratedServiceSetStatusActivityName, isV2)
			a := clusterfeatureworkflow.MakeIntegratedServiceSetStatusActivity(featureRepository)
			worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: activityName})
		}
	}
}
