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

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	clusterfeatureworkflow "github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter/workflow"
)

func registerClusterFeatureWorkflows(featureOperatorRegistry clusterfeature.FeatureOperatorRegistry, featureRepository clusterfeature.FeatureRepository) {
	workflow.RegisterWithOptions(clusterfeatureworkflow.ClusterFeatureJobWorkflow, workflow.RegisterOptions{Name: clusterfeatureworkflow.ClusterFeatureJobWorkflowName})

	{
		a := clusterfeatureworkflow.MakeClusterFeatureApplyActivity(featureOperatorRegistry)
		activity.RegisterWithOptions(a.Execute, activity.RegisterOptions{Name: clusterfeatureworkflow.ClusterFeatureApplyActivityName})
	}

	{
		a := clusterfeatureworkflow.MakeClusterFeatureDeleteActivity(featureRepository)
		activity.RegisterWithOptions(a.Execute, activity.RegisterOptions{Name: clusterfeatureworkflow.ClusterFeatureDeleteActivityName})
	}

	{
		a := clusterfeatureworkflow.MakeClusterFeatureDeactivateActivity(featureOperatorRegistry)
		activity.RegisterWithOptions(a.Execute, activity.RegisterOptions{Name: clusterfeatureworkflow.ClusterFeatureDeactivateActivityName})
	}

	{
		a := clusterfeatureworkflow.MakeClusterFeatureSetSpecActivity(featureRepository)
		activity.RegisterWithOptions(a.Execute, activity.RegisterOptions{Name: clusterfeatureworkflow.ClusterFeatureSetSpecActivityName})
	}

	{
		a := clusterfeatureworkflow.MakeClusterFeatureSetStatusActivity(featureRepository)
		activity.RegisterWithOptions(a.Execute, activity.RegisterOptions{Name: clusterfeatureworkflow.ClusterFeatureSetStatusActivityName})
	}
}
