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

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke"
	pkeawsproviderworkflow "github.com/banzaicloud/pipeline/internal/cluster/distribution/pke/pkeaws/pkeawsprovider/workflow"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke/pkeaws/pkeawsworkflow"
	pkeworkflow "github.com/banzaicloud/pipeline/internal/pke/workflow"
)

func registerPKEWorkflows(
	worker worker.Worker,
	passwordSecrets pkeworkflow.PasswordSecretStore,
	config configuration,
	nodePoolStore pke.NodePoolStore,
	clusterDynamicClientFactory cluster.DynamicClientFactory,
) {
	{
		a := pkeworkflow.NewAssembleHTTPProxySettingsActivity(passwordSecrets)
		worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: pkeworkflow.AssembleHTTPProxySettingsActivityName})
	}

	pkeawsworkflow.NewUpdateNodePoolWorkflow().Register(worker)

	// delete node pool workflow
	pkeawsproviderworkflow.NewDeleteNodePoolWorkflow().Register(worker)

	// node pool delete helper activities
	pkeawsproviderworkflow.NewDeleteStoredNodePoolActivity(nodePoolStore).Register(worker)
}
