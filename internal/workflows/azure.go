// Copyright © 2018 Banzai Cloud
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

package workflows

import (
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/internal/providers/azure"
	intSecret "github.com/banzaicloud/pipeline/internal/secret"
	"github.com/banzaicloud/pipeline/secret"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
)

func registerAzureWorkflows() {
	secretStore := intSecret.NewTypedStore(secret.Store, azure.Provider)
	secrets := intSecret.NewSimpleStore(secretStore)

	createResourceGroupActivity := azure.NewCreateResourceGroupActivity(azure.NewResourceGroupClientFactory(secrets))
	activity.RegisterWithOptions(createResourceGroupActivity.Execute, activity.RegisterOptions{Name: createResourceGroupActivity.Name()})

	createStorageAccountActivity := azure.NewCreateStorageAccountActivity(azure.NewStorageAccountClientFactory(secrets))
	activity.RegisterWithOptions(createStorageAccountActivity.Execute, activity.RegisterOptions{Name: createStorageAccountActivity.Name()})

	createBucketActivity := azure.NewCreateBucketActivity(azure.NewStorageAccountClientFactory(secrets))
	activity.RegisterWithOptions(createBucketActivity.Execute, activity.RegisterOptions{Name: createBucketActivity.Name()})

	createBucketRollbackActivity := azure.NewCreateBucketRollbackActivity(config.DB())
	activity.RegisterWithOptions(createBucketRollbackActivity.Execute, activity.RegisterOptions{Name: createBucketRollbackActivity.Name()})

	createBucketWorkflow := azure.NewCreateBucketWorkflow()
	workflow.RegisterWithOptions(createBucketWorkflow.Execute, workflow.RegisterOptions{Name: createBucketWorkflow.Name()})
}
