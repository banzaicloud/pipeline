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
	BucketID       uint
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
		ScheduleToStartTimeout: 1 * time.Minute,
		StartToCloseTimeout:    1 * time.Minute,
		WaitForCancellation:    true,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          10 * time.Second,
			ExpirationInterval:       30 * time.Minute,
			MaximumAttempts:          5,
			BackoffCoefficient:       1.2,
			NonRetriableErrorReasons: []string{"cadenceInternal:Generic"},
		},
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	rac := CreateResourceGroupActivityContext{
		OrganizationID: workflowContext.OrganizationID,
		SecretID:       workflowContext.SecretID,
		Location:       workflowContext.Location,
		ResourceGroup:  workflowContext.ResourceGroup,
	}

	err := workflow.ExecuteActivity(ctx, CreateResourceGroupActivityType, rac).Get(ctx, nil)
	if err != nil {
		logger.Error("object store bucket creation failed", zap.Error(err))

		return w.rollback(ctx, logger, workflowContext, err)
	}

	sac := CreateStorageAccountActivityContext{
		OrganizationID: workflowContext.OrganizationID,
		SecretID:       workflowContext.SecretID,
		Location:       workflowContext.Location,
		ResourceGroup:  workflowContext.ResourceGroup,
		StorageAccount: workflowContext.StorageAccount,
	}

	err = workflow.ExecuteActivity(ctx, CreateStorageAccountActivityType, sac).Get(ctx, nil)
	if err != nil {
		logger.Error("object store bucket creation failed", zap.Error(err))

		return w.rollback(ctx, logger, workflowContext, err)
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

		return w.rollback(ctx, logger, workflowContext, err)
	}

	logger.Info("object store bucket successfully created")

	return nil
}

func (w *CreateBucketWorkflow) rollback(ctx workflow.Context, logger *zap.Logger, workflowContext CreateBucketWorkflowContext, err error) error {
	activityOptions := workflow.ActivityOptions{
		ScheduleToStartTimeout: 1 * time.Minute,
		StartToCloseTimeout:    1 * time.Minute,
		WaitForCancellation:    true,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          10 * time.Second,
			ExpirationInterval:       10 * time.Minute,
			MaximumAttempts:          100,
			BackoffCoefficient:       1.1,
			NonRetriableErrorReasons: []string{},
		},
	}

	activityContext := CreateBucketRollbackActivityContext{
		BucketID: workflowContext.BucketID,
	}

	rerr := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, activityOptions), CreateBucketRollbackActivityType, activityContext).Get(ctx, nil)
	if rerr != nil {
		logger.Error("object store bucket rollback failed", zap.Error(err))

		return rerr
	}

	return err
}
