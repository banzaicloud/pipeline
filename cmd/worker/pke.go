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

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke/pkeaws/pkeawsprovider/workflow"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke/pkeaws/pkeawsworkflow"
	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
	pkeworkflow "github.com/banzaicloud/pipeline/internal/pke/workflow"
)

func registerPKEWorkflows(
	passwordSecrets pkeworkflow.PasswordSecretStore,
	config configuration,
	secretStore awsworkflow.SecretStore,
	nodePoolStore pke.NodePoolStore,
	clusterDynamicClientFactory cluster.DynamicClientFactory,
) {
	{
		a := pkeworkflow.NewAssembleHTTPProxySettingsActivity(passwordSecrets)
		activity.RegisterWithOptions(a.Execute, activity.RegisterOptions{Name: pkeworkflow.AssembleHTTPProxySettingsActivityName})
	}

	pkeawsworkflow.NewUpdateNodePoolWorkflow().Register()

	awsSessionFactory := awsworkflow.NewAWSSessionFactory(secretStore)

	// delete node pool workflow
	workflow.NewDeleteNodePoolWorkflow().Register()

	// node pool delete helper activities
	deleteStackActivity := awsworkflow.NewDeleteStackActivity(awsSessionFactory)
	activity.RegisterWithOptions(deleteStackActivity.Execute, activity.RegisterOptions{Name: awsworkflow.DeleteStackActivityName})

	deleteStoredNodePoolActivity := workflow.NewDeleteStoredNodePoolActivity(nodePoolStore)
	activity.RegisterWithOptions(deleteStoredNodePoolActivity.Execute, activity.RegisterOptions{
		Name: workflow.DeleteStoredNodePoolActivityName,
	})

	listStoredNodePoolsActivity := workflow.NewListStoredNodePoolsActivity(nodePoolStore)
	activity.RegisterWithOptions(listStoredNodePoolsActivity.Execute, activity.RegisterOptions{
		Name: workflow.ListStoredNodePoolsActivityName,
	})
}
