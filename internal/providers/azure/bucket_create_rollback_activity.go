package azure

import (
	"context"

	"github.com/banzaicloud/pipeline/internal/platform/zaplog"
	"github.com/jinzhu/gorm"
	"go.uber.org/cadence"
	"go.uber.org/cadence/activity"
	"go.uber.org/zap"
)

const CreateBucketRollbackActivityType = "azure-create-bucket-rollback"

type CreateBucketRollbackActivityContext struct {
	BucketID uint
}

type CreateBucketRollbackActivity struct {
	db *gorm.DB
}

func NewCreateBucketRollbackActivity(db *gorm.DB) *CreateBucketRollbackActivity {
	return &CreateBucketRollbackActivity{
		db: db,
	}
}

func (a *CreateBucketRollbackActivity) Name() string {
	return CreateBucketRollbackActivityType
}

func (a *CreateBucketRollbackActivity) Execute(ctx context.Context, activityContext CreateBucketRollbackActivityContext) error {
	logger := activity.GetLogger(ctx).With( // TODO: add correlation ID from API request (if any)
		zap.Uint("bucket-id", activityContext.BucketID),
	)

	logger.Info("deleting bucket state")

	err := DeleteBucketState(a.db, activityContext.BucketID)
	if err != nil {
		zaplog.LogError(logger, err) // TODO: use error handler

		return cadence.NewCustomError("pipeline:DatabaseFailure", "failed to delete bucket state")
	}

	logger.Info("bucket state successfully deleted")

	return nil
}
