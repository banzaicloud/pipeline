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

	cluster2 "github.com/banzaicloud/pipeline/internal/cluster"
	eksworkflow "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke/pkeaws/pkeawsprovider/workflow"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke/pkeaws/pkeawsworkflow"
	pkeworkflow "github.com/banzaicloud/pipeline/internal/pke/workflow"
)

func registerPKEWorkflows(
	passwordSecrets pkeworkflow.PasswordSecretStore,
	config configuration,
	secretStore workflow.SecretStore,
	nodePoolStore pke.NodePoolStore,
	clusterDynamicClientFactory cluster2.DynamicClientFactory,
) {
	{
		a := pkeworkflow.NewAssembleHTTPProxySettingsActivity(passwordSecrets)
		activity.RegisterWithOptions(a.Execute, activity.RegisterOptions{Name: pkeworkflow.AssembleHTTPProxySettingsActivityName})
	}

	pkeawsworkflow.NewUpdateNodePoolWorkflow().Register()

	awsSessionFactory := workflow.NewAWSSessionFactory(secretStore)

	// delete node pool workflow
	workflow.NewDeleteNodePoolWorkflow().Register()

	// node pool delete helper activities
	deleteStackActivity := eksworkflow.NewDeleteStackActivity(awsSessionFactory)
	activity.RegisterWithOptions(deleteStackActivity.Execute, activity.RegisterOptions{Name: eksworkflow.DeleteStackActivityName})

	deleteStoredNodePoolActivity := workflow.NewDeleteStoredNodePoolActivity(nodePoolStore)
	activity.RegisterWithOptions(deleteStoredNodePoolActivity.Execute, activity.RegisterOptions{
		Name: eksworkflow.DeleteStoredNodePoolActivityName,
	})
}
