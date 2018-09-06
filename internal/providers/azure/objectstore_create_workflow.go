package azure

import (
	"time"

	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

const CreateBucketWorkflowType = "azure-create-bucket"

type CreateBucketWorkflowContext struct {
	OrganizationID uint
	SecretID       string
	Location       string
	ResourceGroup  string
	StorageAccount string
	Bucket         string
}
type CreateBucketWorkflow struct{}

func NewCreateBucketWorkflow() *CreateBucketWorkflow {
	return &CreateBucketWorkflow{}
}

func (w *CreateBucketWorkflow) Name() string {
	return CreateBucketWorkflowType
}

func (w *CreateBucketWorkflow) Execute(ctx workflow.Context, workflowContext CreateBucketWorkflowContext) error {
	logger := workflow.GetLogger(ctx).With( // TODO: add correlation ID from API request (if any)
		zap.Uint("organization-id", workflowContext.OrganizationID),
		zap.String("secret-id", workflowContext.SecretID),
		zap.String("location", workflowContext.Location),
		zap.String("resource-group", workflowContext.ResourceGroup),
		zap.String("storage-account", workflowContext.StorageAccount),
		zap.String("bucket", workflowContext.Bucket),
	)

	logger.Info("creating object store bucket")

	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 30 * time.Second,
		StartToCloseTimeout:    1 * time.Minute,
		WaitForCancellation:    true,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          3 * time.Second,
			ExpirationInterval:       3 * time.Minute,
			MaximumAttempts:          3,
			BackoffCoefficient:       1.0,
			NonRetriableErrorReasons: []string{"cadenceInternal:Generic"},
		},
	}

	rac := CreateResourceGroupActivityContext{
		OrganizationID: workflowContext.OrganizationID,
		SecretID:       workflowContext.SecretID,
		Location:       workflowContext.Location,
		ResourceGroup:  workflowContext.ResourceGroup,
	}

	err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, ao), CreateResourceGroupActivityType, rac).Get(ctx, nil)
	if err != nil {
		logger.Error("object store bucket creation failed", zap.Error(err))

		return err
	}

	//ao.StartToCloseTimeout = 10 * time.Minute

	sac := CreateStorageAccountActivityContext{
		OrganizationID: workflowContext.OrganizationID,
		SecretID:       workflowContext.SecretID,
		Location:       workflowContext.Location,
		ResourceGroup:  workflowContext.ResourceGroup,
		StorageAccount: workflowContext.StorageAccount,
	}

	err = workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, ao), CreateStorageAccountActivityType, sac).Get(ctx, nil)
	if err != nil {
		logger.Error("object store bucket creation failed", zap.Error(err))

		return err
	}

	bac := CreateBucketActivityContext{
		OrganizationID: workflowContext.OrganizationID,
		SecretID:       workflowContext.SecretID,
		ResourceGroup:  workflowContext.ResourceGroup,
		StorageAccount: workflowContext.StorageAccount,
		Bucket:         workflowContext.Bucket,
	}

	err = workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, ao), CreateBucketActivityType, bac).Get(ctx, nil)
	if err != nil {
		logger.Error("object store bucket creation failed", zap.Error(err))

		return err
	}

	logger.Info("object store bucket successfully created")

	return nil
}
