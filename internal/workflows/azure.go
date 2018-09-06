package workflows

import (
	"github.com/banzaicloud/pipeline/internal/providers/azure"
	intSecret "github.com/banzaicloud/pipeline/internal/secret"
	"github.com/banzaicloud/pipeline/secret"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
)

func registerAzureWorkflows() {
	secretStore := intSecret.NewTypedStore(secret.Store, azure.Provider)
	secrets := intSecret.NewSimpleStore(secretStore)

	createResourceGroupActivitiy := azure.NewCreateResourceGroupActivity(azure.NewResourceGroupClientFactory(secrets))
	activity.RegisterWithOptions(createResourceGroupActivitiy.Execute, activity.RegisterOptions{Name: createResourceGroupActivitiy.Name()})

	createStorageAccountActivitiy := azure.NewCreateStorageAccountActivity(azure.NewStorageAccountClientFactory(secrets))
	activity.RegisterWithOptions(createStorageAccountActivitiy.Execute, activity.RegisterOptions{Name: createStorageAccountActivitiy.Name()})

	createBucketActivitiy := azure.NewCreateBucketActivity(azure.NewStorageAccountClientFactory(secrets))
	activity.RegisterWithOptions(createBucketActivitiy.Execute, activity.RegisterOptions{Name: createBucketActivitiy.Name()})

	createBucketWorkflow := azure.NewCreateBucketWorkflow()
	workflow.RegisterWithOptions(createBucketWorkflow.Execute, workflow.RegisterOptions{Name: createBucketWorkflow.Name()})
}
